package application

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"stage-rigging-clearance/internal/assessment"
	"stage-rigging-clearance/internal/domain"
	"stage-rigging-clearance/internal/storage"
	"strings"
	"sync"
	"time"
)

type Service struct {
	store *storage.Store
	mu    sync.Mutex
}

func New(s *storage.Store) *Service { return &Service{store: s} }
func newID(prefix string) string    { return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()) }
func (s *Service) Create(performance, zones, owner, date string) (*domain.RiggingCase, error) {
	issues := domain.ValidateCaseInput(performance, zones, owner)
	d, dateIssues := domain.ValidatePerformanceDate(date, time.Now())
	issues = append(issues, dateIssues...)
	if len(issues) > 0 {
		return nil, domain.ValidationError{Issues: issues}
	}
	c := domain.NewCase(newID("case"), performance, zones, owner, d)
	if e := c.StartPlanning(strings.TrimSpace(owner)); e != nil {
		return nil, e
	}
	if e := s.store.Save(c, 1); e != nil {
		return nil, e
	}
	return c, nil
}
func (s *Service) UpdateCase(id string, expected int, performance, zones, owner, date, actor string) (*domain.RiggingCase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, e := s.store.Get(id)
	if e != nil {
		return nil, e
	}
	if c.ExpectedVersion != expected {
		return nil, domain.ErrVersionConflict
	}
	if c.Status == domain.StatusFrozen {
		return nil, domain.ErrFrozen
	}
	issues := domain.ValidateCaseInput(performance, zones, owner)
	d, ds := domain.ValidatePerformanceDate(date, time.Now())
	issues = append(issues, ds...)
	if len(issues) > 0 {
		return nil, domain.ValidationError{Issues: issues}
	}
	c.PerformanceName, c.StageZones, c.OwnerName, c.PerformanceDate = strings.TrimSpace(performance), strings.TrimSpace(zones), strings.TrimSpace(owner), d
	c.LastAssessmentDigest = ""
	c.LastAssessment = nil
	c.Reviews = nil
	if c.CurrentRevision() != nil {
		c.Status = domain.StatusPlanning
	}
	c.ExpectedVersion++
	c.UpdatedAt = time.Now().UTC()
	c.Audit = append(c.Audit, domain.AuditEvent{Seq: len(c.Audit) + 1, Type: "CaseUpdated", Message: "任务资料更新", Actor: strings.TrimSpace(actor), At: c.UpdatedAt})
	if e = s.store.Save(c, expected); e != nil {
		return nil, e
	}
	return c, nil
}
func (s *Service) SubmitRevision(id string, expected int, note, by string, points []domain.LoadPoint) (*domain.RiggingCase, error) {
	return s.SubmitRevisionWithKey(id, expected, note, by, "", points)
}
func (s *Service) SubmitRevisionWithKey(id string, expected int, note, by, key string, points []domain.LoadPoint) (*domain.RiggingCase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key = strings.TrimSpace(key)
	c, e := s.store.Get(id)
	if e != nil {
		return nil, e
	}
	if strings.TrimSpace(key) != "" {
		if rid := c.RevisionIdempotency[key]; rid != "" {
			return c, nil
		}
	}
	if c.ExpectedVersion != expected {
		return nil, domain.ErrVersionConflict
	}
	if issues := domain.ValidateRevision(domain.PlanRevision{Points: points}); len(issues) > 0 {
		return nil, domain.ValidationError{Issues: issues}
	}
	r := domain.PlanRevision{ID: newID("revision"), ChangeNote: note, SubmittedBy: by, SubmittedAt: time.Now().UTC(), Points: points, ReplacesRevisionID: c.CurrentRevisionID}
	if e = c.AddRevision(r, by); e != nil {
		return nil, e
	}
	if c.RevisionIdempotency == nil {
		c.RevisionIdempotency = map[string]string{}
	}
	if strings.TrimSpace(key) != "" {
		c.RevisionIdempotency[key] = r.ID
	}
	if e = s.store.Save(c, expected); e != nil {
		return nil, e
	}
	return c, nil
}
func (s *Service) Assess(id string) (assessment.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, e := s.store.Get(id)
	if e != nil {
		return assessment.Result{}, e
	}
	if c.Status == domain.StatusFrozen {
		return assessment.Result{}, domain.ErrFrozen
	}
	r := c.CurrentRevision()
	if r == nil {
		return assessment.Result{}, domain.ErrInvalidState
	}
	res := assessment.Evaluate(c, r)
	c.ReplaceCurrentFindings(res.Findings)
	if e = c.SetValidation(!res.HasBlocking(), res.InputDigest, "核验引擎"); e != nil {
		return assessment.Result{}, e
	}
	snap := &domain.AssessmentSnapshot{RevisionID: res.RevisionID, InputDigest: res.InputDigest, RuleVersion: res.RuleVersion, ZoneLoads: res.ZoneLoads, Eccentricity: res.Eccentricity, PointMetrics: []domain.AssessmentPointMetric{}, ZoneMetrics: []domain.AssessmentZoneMetric{}, FindingIDs: []string{}}
	for _, p := range res.PointMetrics {
		snap.PointMetrics = append(snap.PointMetrics, domain.AssessmentPointMetric{PointCode: p.PointCode, Utilization: p.Utilization})
	}
	for _, z := range res.ZoneMetrics {
		snap.ZoneMetrics = append(snap.ZoneMetrics, domain.AssessmentZoneMetric{Zone: z.Zone, LoadKg: z.LoadKg, PointCount: z.PointCount})
	}
	for _, f := range res.Findings {
		snap.FindingIDs = append(snap.FindingIDs, f.ID)
	}
	c.LastAssessment = snap
	if e = s.store.Save(c, c.ExpectedVersion-1); e != nil {
		return assessment.Result{}, e
	}
	return res, nil
}
func (s *Service) Resolve(id, findingID, revisionID, note, by string, expected int) (*domain.RiggingCase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, e := s.store.Get(id)
	if e != nil {
		return nil, e
	}
	if c.ExpectedVersion != expected {
		return nil, domain.ErrVersionConflict
	}
	if revisionID != c.CurrentRevisionID || strings.TrimSpace(note) == "" {
		return nil, domain.ErrInvalidInput
	}
	finding, e := c.FindingByID(findingID)
	if e != nil {
		return nil, e
	}
	if finding.RemediationRevisionID != revisionID || strings.TrimSpace(finding.RemediationNote) == "" {
		if e = c.AssociateRemediation(findingID, revisionID, note); e != nil {
			return nil, e
		}
		c.Record("RemediationAssociated", by)
	}
	if e = c.ResolveFinding(findingID, by, note); e != nil {
		return nil, e
	}
	if e = s.store.Save(c, expected); e != nil {
		return nil, e
	}
	return c, nil
}
func (s *Service) AssociateRemediation(id, findingID, revisionID, note, by string, expected int) (*domain.RiggingCase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, e := s.store.Get(id)
	if e != nil {
		return nil, e
	}
	if c.ExpectedVersion != expected {
		return nil, domain.ErrVersionConflict
	}
	if e = c.AssociateRemediation(findingID, revisionID, note); e != nil {
		return nil, e
	}
	c.Record("RemediationAssociated", by)
	if e = s.store.Save(c, expected); e != nil {
		return nil, e
	}
	return c, nil
}
func (s *Service) Reopen(id, findingID, by string, expected int) (*domain.RiggingCase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, e := s.store.Get(id)
	if e != nil {
		return nil, e
	}
	if c.ExpectedVersion != expected {
		return nil, domain.ErrVersionConflict
	}
	if e = c.ReopenFinding(findingID, by); e != nil {
		return nil, e
	}
	if e = s.store.Save(c, expected); e != nil {
		return nil, e
	}
	return c, nil
}
func (s *Service) Review(id string, expected int, stage, outcome, reviewer, comment string) (*domain.RiggingCase, error) {
	c, e := s.Get(id)
	if e != nil {
		return nil, e
	}
	r := c.CurrentRevision()
	if r == nil {
		return nil, domain.ErrInvalidState
	}
	return s.ReviewWithDigest(id, expected, stage, outcome, reviewer, comment, r.ContentDigest)
}
func (s *Service) ReviewWithDigest(id string, expected int, stage, outcome, reviewer, comment, digest string) (*domain.RiggingCase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, e := s.store.Get(id)
	if e != nil {
		return nil, e
	}
	if c.ExpectedVersion != expected {
		return nil, domain.ErrVersionConflict
	}
	r := c.CurrentRevision()
	if r == nil {
		return nil, domain.ErrInvalidState
	}
	if c.LastAssessmentDigest != r.ContentDigest {
		return nil, domain.ErrStaleAssessment
	}
	if strings.TrimSpace(digest) == "" || digest != r.ContentDigest {
		return nil, domain.ErrStaleAssessment
	}
	d := domain.ReviewDecision{Stage: domain.ReviewStage(stage), Outcome: domain.ReviewOutcome(outcome), Reviewer: reviewer, Comment: comment, RevisionID: r.ID, RevisionDigest: r.ContentDigest}
	if e = c.AddReview(d, reviewer); e != nil {
		return nil, e
	}
	if e = s.store.Save(c, expected); e != nil {
		return nil, e
	}
	return c, nil
}
func (s *Service) Freeze(id string, expected int, by string) (*domain.ClearancePermit, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, e := s.store.Get(id)
	if e != nil {
		return nil, e
	}
	if c.ExpectedVersion != expected {
		return nil, domain.ErrVersionConflict
	}
	p, e := c.Freeze(by)
	if e != nil {
		return nil, e
	}
	if e = s.store.Save(c, expected); e != nil {
		return nil, e
	}
	return &p, nil
}
func (s *Service) Get(id string) (*domain.RiggingCase, error) { return s.store.Get(id) }
func (s *Service) List() ([]*domain.RiggingCase, error) {
	ids := s.store.CaseIDs()
	out := make([]*domain.RiggingCase, 0, len(ids))
	for _, id := range ids {
		c, e := s.store.Get(id)
		if e != nil {
			return nil, e
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
func (s *Service) Verify(code string) (bool, *domain.ClearancePermit, error) {
	c, e := s.store.FindPermit(code)
	if e != nil {
		return false, nil, e
	}
	if c.Permit == nil {
		return false, nil, domain.ErrNotFound
	}
	r := c.CurrentRevision()
	if r == nil {
		return false, nil, domain.ErrNotFound
	}
	snapshot := domain.SnapshotDigest(c, r)
	vh := sha256.Sum256([]byte(code + snapshot))
	valid := snapshot == c.Permit.SnapshotDigest && hex.EncodeToString(vh[:]) == c.Permit.VerificationDigest
	return valid, c.Permit, nil
}
