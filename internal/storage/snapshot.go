package storage

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"stage-rigging-clearance/internal/domain"
)

type Snapshot struct {
	Cases   map[string]*domain.RiggingCase `json:"cases"`
	Permits map[string]string              `json:"permits"`
	Version int                            `json:"version"`
}

func SnapshotDigest(c *domain.RiggingCase) string {
	b, _ := json.Marshal(c)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (s *Store) ReadEvents() ([]Event, error) {
	if s.dir == "" {
		return nil, nil
	}
	f, e := os.Open(filepath.Join(s.dir, "events.jsonl"))
	if os.IsNotExist(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	defer f.Close()
	var out []Event
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var x Event
		if json.Unmarshal(sc.Bytes(), &x) != nil || ValidateEvent(x) != nil {
			return nil, domain.ErrInvalidInput
		}
		out = append(out, x)
	}
	return out, sc.Err()
}
func (s *Store) ValidateLog() error {
	events, e := s.ReadEvents()
	if e != nil {
		return fmt.Errorf("事件日志恢复失败：%w", e)
	}
	last := 0
	for _, x := range events {
		if x.Seq != last+1 {
			return fmt.Errorf("事件日志恢复失败：序号不连续（期望 %d，实际 %d）：%w", last+1, x.Seq, domain.ErrInvalidInput)
		}
		if x.Digest == "" && (x.PrevDigest != "" || s.lastDigest != "") {
			return fmt.Errorf("事件日志恢复失败：摘要缺失：%w", domain.ErrInvalidInput)
		}
		if last == 0 && x.PrevDigest != "" {
			return fmt.Errorf("事件日志恢复失败：首事件摘要链无效：%w", domain.ErrInvalidInput)
		}
		if last > 0 && x.PrevDigest != "" && x.PrevDigest != s.lastDigest {
			return fmt.Errorf("事件日志恢复失败：摘要链不一致：%w", domain.ErrInvalidInput)
		}
		if _, ok := s.cases[x.CaseID]; !ok {
			return fmt.Errorf("事件日志恢复失败：任务不存在（%s）：%w", x.CaseID, domain.ErrInvalidInput)
		}
		last = x.Seq
		s.lastDigest = x.Digest
	}
	if s.seq > 0 && last != s.seq {
		return fmt.Errorf("事件日志恢复失败：快照版本 %d 与事件序号 %d 不一致：%w", s.seq, last, domain.ErrInvalidInput)
	}
	s.mu.Lock()
	if last > s.seq {
		s.seq = last
	}
	s.mu.Unlock()
	return nil
}
func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cases := make(map[string]*domain.RiggingCase, len(s.cases))
	for id, c := range s.cases {
		var cp domain.RiggingCase
		b, _ := json.Marshal(c)
		_ = json.Unmarshal(b, &cp)
		cases[id] = &cp
	}
	permits := make(map[string]string, len(s.permits))
	for k, v := range s.permits {
		permits[k] = v
	}
	return Snapshot{Cases: cases, Permits: permits, Version: s.seq}
}
func (s *Store) CaseIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.cases))
	for id := range s.cases {
		out = append(out, id)
	}
	return out
}
