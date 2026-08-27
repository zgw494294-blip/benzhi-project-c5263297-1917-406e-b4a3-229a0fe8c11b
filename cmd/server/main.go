package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"stage-rigging-clearance/internal/application"
	"stage-rigging-clearance/internal/storage"
	"stage-rigging-clearance/internal/web"
	"strings"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:19081", "回环监听地址")
	self := flag.Bool("selfcheck", false, "运行完整回环自检")
	flag.Parse()
	addrSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "addr" {
			addrSet = true
		}
	})
	if env := os.Getenv("PORT"); env != "" && !addrSet {
		*addr = "127.0.0.1:" + env
	}
	if !strings.HasPrefix(*addr, "127.0.0.1:") {
		fmt.Fprintln(os.Stderr, "监听地址必须为回环地址")
		os.Exit(2)
	}
	st, e := storage.New(".benzhi/data")
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	srv := http.Server{Addr: *addr, Handler: web.New(application.New(st)).Mux}
	if *self {
		if e := runSelfCheck(&srv, *addr); e != nil {
			fmt.Fprintln(os.Stderr, "自检失败:", e)
			os.Exit(1)
		}
		fmt.Println("自检通过")
		return
	}
	fmt.Println("服务监听", *addr)
	if e := srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func runSelfCheck(srv *http.Server, addr string) error {
	ln, e := net.Listen("tcp", addr)
	if e != nil {
		return e
	}
	go srv.Serve(ln)
	defer func() {
		ctx, c := context.WithTimeout(context.Background(), time.Second)
		_ = srv.Shutdown(ctx)
		if c != nil {
			c()
		}
	}()
	base := "http://" + ln.Addr().String()
	page, e := http.Get(base + "/")
	if e != nil {
		return e
	}
	if page.StatusCode != http.StatusOK {
		page.Body.Close()
		return fmt.Errorf("工作台页面状态 %s", page.Status)
	}
	page.Body.Close()
	var c struct {
		ID              string `json:"caseId"`
		ExpectedVersion int
	}
	if e = post(base+"/api/cases", map[string]any{"performanceName": "自检演出", "stageZones": "主舞台", "ownerName": "自检员", "performanceDate": time.Now().UTC().Format("2006-01-02")}, &c); e != nil {
		return e
	}
	var rev struct {
		ExpectedVersion   int
		CurrentRevisionID string
		Revisions         []struct {
			ID            string `json:"revisionId"`
			ContentDigest string `json:"contentDigest"`
		} `json:"revisions"`
	}
	if e = post(base+"/api/cases/"+c.ID+"/revisions", map[string]any{"expectedVersion": c.ExpectedVersion, "note": "自检初版", "by": "自检员", "points": []map[string]any{{"pointCode": "A", "stageZone": "主舞台", "componentName": "吊杆", "ratedLoadKg": 1000, "actualLoadKg": 500, "clearanceMm": 800, "cueStart": 1, "cueEnd": 10}}}, &rev); e != nil {
		return e
	}
	var revisionDigest string
	for _, item := range rev.Revisions {
		if item.ID == rev.CurrentRevisionID {
			revisionDigest = item.ContentDigest
		}
	}
	if revisionDigest == "" {
		return fmt.Errorf("自检未读取修订摘要")
	}
	var assess map[string]any
	if e = post(base+"/api/cases/"+c.ID+"/assess", "{}", &assess); e != nil {
		return e
	}
	var state struct {
		ExpectedVersion   int
		CurrentRevisionID string
	}
	if e = get(base+"/api/cases/"+c.ID, &state); e != nil {
		return e
	}
	var out map[string]any
	if e = post(base+"/api/cases/"+c.ID+"/reviews", map[string]any{"expectedVersion": state.ExpectedVersion, "stage": "Safety", "outcome": "Approve", "reviewer": "安全员", "comment": "通过", "revisionDigest": revisionDigest}, &out); e != nil {
		return e
	}
	if e = get(base+"/api/cases/"+c.ID, &state); e != nil {
		return e
	}
	if e = post(base+"/api/cases/"+c.ID+"/reviews", map[string]any{"expectedVersion": state.ExpectedVersion, "stage": "Technical", "outcome": "Approve", "reviewer": "技术负责人", "comment": "通过", "revisionDigest": revisionDigest}, &out); e != nil {
		return e
	}
	if e = get(base+"/api/cases/"+c.ID, &state); e != nil {
		return e
	}
	var permit struct{ PermitCode string }
	if e = post(base+"/api/cases/"+c.ID+"/freeze", map[string]any{"expectedVersion": state.ExpectedVersion, "by": "技术负责人"}, &permit); e != nil {
		return e
	}
	if e = get(base+"/api/permits/"+permit.PermitCode, &out); e != nil {
		return e
	}
	return nil
}
func post(url string, body any, out any) error {
	var b []byte
	switch x := body.(type) {
	case string:
		b = []byte(x)
	default:
		b, _ = json.Marshal(x)
	}
	r, e := http.Post(url, "application/json", bytes.NewReader(b))
	if e != nil {
		return e
	}
	defer r.Body.Close()
	if r.StatusCode >= 300 {
		x, _ := io.ReadAll(r.Body)
		return fmt.Errorf("%s %s: %s", r.Status, url, string(x))
	}
	return json.NewDecoder(r.Body).Decode(out)
}
func get(url string, out any) error {
	r, e := http.Get(url)
	if e != nil {
		return e
	}
	defer r.Body.Close()
	if r.StatusCode >= 300 {
		return fmt.Errorf("get %s", r.Status)
	}
	return json.NewDecoder(r.Body).Decode(out)
}
