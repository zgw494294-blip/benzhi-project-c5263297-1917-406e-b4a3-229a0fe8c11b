package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"stage-rigging-clearance/internal/application"
	"stage-rigging-clearance/internal/domain"
	"strings"
)

type Server struct {
	App *application.Service
	Mux *http.ServeMux
}

func New(app *application.Service) *Server {
	s := &Server{App: app, Mux: http.NewServeMux()}
	s.routes()
	return s
}
func (s *Server) routes() {
	s.Mux.HandleFunc("/style.css", AssetHandler)
	s.Mux.HandleFunc("/app.js", AssetHandler)
	s.Mux.HandleFunc("/", s.home)
	s.Mux.HandleFunc("/api/cases", s.cases)
	s.Mux.HandleFunc("/api/cases/", s.caseAction)
	s.Mux.HandleFunc("/api/permits/", s.verify)
}
func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHTML))
}
func jsonOut(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
func errOut(w http.ResponseWriter, e error) {
	code := http.StatusBadRequest
	if e == domain.ErrNotFound {
		code = http.StatusNotFound
	}
	if e == domain.ErrVersionConflict {
		code = http.StatusConflict
	}
	classified := classifyError(e)
	resp := map[string]any{"error": e.Error(), "code": classified.Code, "retryable": classified.Retryable}
	if v, ok := e.(domain.ValidationError); ok {
		resp["issues"] = v.Issues
	}
	writeJSON(w, code, resp)
}
func (s *Server) caseError(w http.ResponseWriter, id string, e error) {
	if e == domain.ErrVersionConflict {
		if c, ge := s.App.Get(id); ge == nil {
			classified := classifyError(e)
			writeJSON(w, http.StatusConflict, map[string]any{"error": e.Error(), "code": classified.Code, "retryable": classified.Retryable, "latestVersion": c.ExpectedVersion})
			return
		}
	}
	errOut(w, e)
}
func (s *Server) cases(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		cs, e := s.App.List()
		if e != nil {
			errOut(w, e)
			return
		}
		out := make([]map[string]any, 0, len(cs))
		for _, c := range cs {
			out = append(out, casePayload(c))
		}
		jsonOut(w, out)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "请求方法不支持", http.StatusMethodNotAllowed)
		return
	}
	var q struct{ PerformanceName, StageZones, OwnerName, PerformanceDate string }
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		errOut(w, domain.ErrInvalidInput)
		return
	}
	c, e := s.App.Create(q.PerformanceName, q.StageZones, q.OwnerName, q.PerformanceDate)
	if e != nil {
		errOut(w, e)
		return
	}
	jsonOut(w, c)
}
func (s *Server) caseAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}
	id := parts[2]
	if len(parts) == 3 && r.Method == "GET" {
		c, e := s.App.Get(id)
		if e != nil {
			errOut(w, e)
			return
		}
		jsonOut(w, casePayload(c))
		return
	}
	if len(parts) == 3 && (r.Method == http.MethodPut || r.Method == http.MethodPatch) {
		s.updateCase(w, r, id)
		return
	}
	if len(parts) == 4 && r.Method == "GET" && parts[3] == "timeline" {
		t, e := s.App.Timeline(id)
		if e != nil {
			errOut(w, e)
			return
		}
		jsonOut(w, t)
		return
	}
	if len(parts) == 4 && r.Method == "GET" && parts[3] == "export" {
		b, e := s.App.Export(id)
		if e != nil {
			errOut(w, e)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
		return
	}
	if len(parts) == 4 && r.Method == "GET" && parts[3] == "revisions" {
		c, e := s.App.Get(id)
		if e != nil {
			errOut(w, e)
			return
		}
		jsonOut(w, c.RevisionHistory())
		return
	}
	if len(parts) < 4 {
		http.NotFound(w, r)
		return
	}
	action := parts[3]
	switch action {
	case "revisions":
		if len(parts) > 4 && parts[4] == "compare" {
			s.compare(w, r, id)
			return
		}
		s.revision(w, r, id)
	case "assess":
		if len(parts) == 4 && r.Method == http.MethodGet {
			c, e := s.App.Get(id)
			if e != nil {
				errOut(w, e)
				return
			}
			jsonOut(w, assessmentPayload(c, r))
			return
		}
		s.assess(w, r, id)
	case "assessment":
		if r.Method != http.MethodGet {
			http.Error(w, "请求方法不支持", http.StatusMethodNotAllowed)
			return
		}
		c, e := s.App.Get(id)
		if e != nil {
			errOut(w, e)
			return
		}
		jsonOut(w, assessmentPayload(c, r))
	case "findings":
		if len(parts) > 4 {
			s.resolve(w, r, id, parts[4])
		} else if r.Method == http.MethodGet {
			fs, e := s.App.Findings(id)
			if e != nil {
				errOut(w, e)
				return
			}
			jsonOut(w, fs)
		}
	case "reviews":
		s.review(w, r, id)
	case "freeze":
		s.freeze(w, r, id)
	case "compare":
		s.compare(w, r, id)
	default:
		http.NotFound(w, r)
	}
}
func (s *Server) updateCase(w http.ResponseWriter, r *http.Request, id string) {
	var q struct {
		ExpectedVersion                                             int `json:"expectedVersion"`
		PerformanceName, StageZones, OwnerName, PerformanceDate, By string
	}
	if decodeBody(r, &q) != nil {
		errOut(w, domain.ErrInvalidInput)
		return
	}
	c, e := s.App.UpdateCase(id, q.ExpectedVersion, q.PerformanceName, q.StageZones, q.OwnerName, q.PerformanceDate, q.By)
	if e != nil {
		s.caseError(w, id, e)
		return
	}
	jsonOut(w, c)
}
func casePayload(c *domain.RiggingCase) map[string]any {
	b, _ := json.Marshal(c)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	out["statusLabel"] = domain.StateLabel(c.Status)
	out["dataComplete"] = len(domain.ValidateCaseInput(c.PerformanceName, c.StageZones, c.OwnerName)) == 0 && !c.PerformanceDate.IsZero()
	out["assessmentFresh"] = c.CurrentRevision() != nil && c.LastAssessmentDigest == c.CurrentRevision().ContentDigest
	return out
}
func assessmentPayload(c *domain.RiggingCase, r *http.Request) map[string]any {
	if c.LastAssessment == nil {
		return map[string]any{"assessment": nil, "findings": []domain.SafetyFinding{}}
	}
	b, _ := json.Marshal(c.LastAssessment)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	zone, severity, point := strings.TrimSpace(r.URL.Query().Get("zone")), strings.TrimSpace(r.URL.Query().Get("severity")), strings.TrimSpace(r.URL.Query().Get("pointCode"))
	findings := make([]domain.SafetyFinding, 0)
	for _, f := range c.Findings {
		if f.RevisionID != c.CurrentRevisionID {
			continue
		}
		if severity != "" && string(f.Severity) != severity && domain.SeverityLabel(f.Severity) != severity {
			continue
		}
		if point != "" {
			hit := false
			for _, p := range f.RelatedPointIDs {
				if p == point {
					hit = true
				}
			}
			if !hit {
				continue
			}
		}
		findings = append(findings, f)
	}
	if zone != "" {
		if z, ok := out["zoneMetrics"].([]any); ok {
			filtered := make([]any, 0)
			for _, item := range z {
				if m, ok := item.(map[string]any); ok && m["zone"] == zone {
					filtered = append(filtered, item)
				}
			}
			out["zoneMetrics"] = filtered
		}
		if loads, ok := out["zoneLoads"].(map[string]any); ok {
			filtered := map[string]any{}
			if v, found := loads[zone]; found {
				filtered[zone] = v
			}
			out["zoneLoads"] = filtered
		}
	}
	if point != "" {
		if ps, ok := out["pointMetrics"].([]any); ok {
			filtered := make([]any, 0)
			for _, item := range ps {
				if m, ok := item.(map[string]any); ok && m["pointCode"] == point {
					filtered = append(filtered, item)
				}
			}
			out["pointMetrics"] = filtered
		}
	}
	out["findings"] = findings
	return out
}
func (s *Server) revision(w http.ResponseWriter, r *http.Request, id string) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var q struct {
		ExpectedVersion          int `json:"expectedVersion"`
		Note, By, IdempotencyKey string
		Points                   []domain.LoadPoint
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		errOut(w, domain.ErrInvalidInput)
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		key = q.IdempotencyKey
	}
	c, e := s.App.SubmitRevisionWithKey(id, q.ExpectedVersion, q.Note, q.By, key, q.Points)
	if e != nil {
		s.caseError(w, id, e)
		return
	}
	jsonOut(w, c)
}
func (s *Server) assess(w http.ResponseWriter, r *http.Request, id string) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	x, e := s.App.Assess(id)
	if e != nil {
		s.caseError(w, id, e)
		return
	}
	jsonOut(w, x)
}
func (s *Server) resolve(w http.ResponseWriter, r *http.Request, id, fid string) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var q struct {
		ExpectedVersion                                                                                   int
		RevisionID, RemediationRevisionID, Note, RemediationNote, Comment, By, VerifiedBy, Action, Status string
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		errOut(w, domain.ErrInvalidInput)
		return
	}
	if q.RevisionID == "" {
		q.RevisionID = q.RemediationRevisionID
	}
	if q.Note == "" {
		q.Note = q.RemediationNote
	}
	if q.Note == "" {
		q.Note = q.Comment
	}
	if q.By == "" {
		q.By = q.VerifiedBy
	}
	if strings.EqualFold(q.Action, "reopen") || strings.EqualFold(q.Status, "open") {
		if q.By == "" {
			q.By = "复核员"
		}
		c, e := s.App.Reopen(id, fid, q.By, q.ExpectedVersion)
		if e != nil {
			s.caseError(w, id, e)
			return
		}
		jsonOut(w, c)
		return
	}
	if strings.EqualFold(q.Action, "associate") || (q.By == "" && q.VerifiedBy == "" && q.Note != "") {
		c, e := s.App.AssociateRemediation(id, fid, q.RevisionID, q.Note, q.By, q.ExpectedVersion)
		if e != nil {
			s.caseError(w, id, e)
			return
		}
		jsonOut(w, c)
		return
	}
	c, e := s.App.Resolve(id, fid, q.RevisionID, q.Note, q.By, q.ExpectedVersion)
	if e != nil {
		s.caseError(w, id, e)
		return
	}
	jsonOut(w, c)
}
func (s *Server) review(w http.ResponseWriter, r *http.Request, id string) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var q struct {
		ExpectedVersion                                             int
		Stage, Outcome, Decision, Reviewer, Comment, RevisionDigest string
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		errOut(w, domain.ErrInvalidInput)
		return
	}
	if q.Outcome == "" {
		q.Outcome = q.Decision
	}
	c, e := s.App.ReviewWithDigest(id, q.ExpectedVersion, q.Stage, q.Outcome, q.Reviewer, q.Comment, q.RevisionDigest)
	if e != nil {
		s.caseError(w, id, e)
		return
	}
	jsonOut(w, c)
}
func (s *Server) freeze(w http.ResponseWriter, r *http.Request, id string) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var q struct {
		ExpectedVersion int
		By, IssuedBy    string
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		errOut(w, domain.ErrInvalidInput)
		return
	}
	if q.By == "" {
		q.By = q.IssuedBy
	}
	p, e := s.App.Freeze(id, q.ExpectedVersion, q.By)
	if e != nil {
		s.caseError(w, id, e)
		return
	}
	jsonOut(w, p)
}
func (s *Server) compare(w http.ResponseWriter, r *http.Request, id string) {
	a, b := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if a == "" {
		a = r.URL.Query().Get("fromRevisionId")
	}
	if a == "" {
		a = r.URL.Query().Get("revisionA")
	}
	if a == "" {
		a = r.URL.Query().Get("a")
	}
	if b == "" {
		b = r.URL.Query().Get("toRevisionId")
	}
	if b == "" {
		b = r.URL.Query().Get("revisionB")
	}
	if b == "" {
		b = r.URL.Query().Get("b")
	}
	if (a == "" || b == "") && r.Method == http.MethodPost {
		var q struct{ From, To, FromRevisionID, ToRevisionID string }
		if decodeBody(r, &q) == nil {
			a, b = q.From, q.To
			if a == "" {
				a = q.FromRevisionID
			}
			if b == "" {
				b = q.ToRevisionID
			}
		}
	}
	if a == "" || b == "" {
		errOut(w, fmt.Errorf("请输入 from 和 to"))
		return
	}
	d, e := s.App.Compare(id, a, b)
	if e != nil {
		errOut(w, e)
		return
	}
	jsonOut(w, d)
}
func (s *Server) verify(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/api/permits/")
	ok, p, e := s.App.Verify(code)
	if e != nil {
		errOut(w, e)
		return
	}
	jsonOut(w, map[string]any{"valid": ok, "permit": p})
}
