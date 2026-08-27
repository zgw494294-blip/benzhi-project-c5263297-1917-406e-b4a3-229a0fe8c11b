package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func NewCase(id, performance, zones, owner string, date time.Time) *RiggingCase {
	now := time.Now().UTC()
	return &RiggingCase{ID: id, PerformanceName: strings.TrimSpace(performance), StageZones: strings.TrimSpace(zones), OwnerName: strings.TrimSpace(owner), PerformanceDate: date.UTC(), Status: StatusDraft, ExpectedVersion: 1, CreatedAt: now, UpdatedAt: now, RevisionIdempotency: map[string]string{}}
}
func (c *RiggingCase) StartPlanning(actor string) error {
	if c.Status != StatusDraft {
		return ErrInvalidState
	}
	c.Status = StatusPlanning
	c.touch("Planning", actor)
	return nil
}
func (c *RiggingCase) touch(msg, actor string) {
	c.ExpectedVersion++
	c.UpdatedAt = time.Now().UTC()
	c.Audit = append(c.Audit, AuditEvent{Seq: len(c.Audit) + 1, Type: msg, Message: msg, Actor: strings.TrimSpace(actor), At: c.UpdatedAt})
}
func (c *RiggingCase) CurrentRevision() *PlanRevision {
	for i := range c.Revisions {
		if c.Revisions[i].ID == c.CurrentRevisionID {
			return &c.Revisions[i]
		}
	}
	return nil
}
func DigestPoints(points []LoadPoint) string {
	var b strings.Builder
	for _, p := range points {
		fmt.Fprintf(&b, "%s|%s|%s|%.2f|%.2f|%.2f|%.2f|%.2f|%d|%d;", p.PointCode, p.StageZone, p.ComponentName, p.RatedLoadKg, p.ActualLoadKg, p.ClearanceMm, p.PositionXmm, p.PositionYmm, p.CueStart, p.CueEnd)
	}
	h := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(h[:])
}
func (c *RiggingCase) AddRevision(r PlanRevision, actor string) error {
	r.Points = append([]LoadPoint(nil), r.Points...)
	r.ChangeNote = strings.TrimSpace(r.ChangeNote)
	r.SubmittedBy = strings.TrimSpace(r.SubmittedBy)
	if c.Status == StatusFrozen {
		return ErrFrozen
	}
	if c.Status != StatusPlanning && c.Status != StatusValidationFailed && c.Status != StatusRemediating && c.Status != StatusChangesRequested && c.Status != StatusUnderReview && c.Status != StatusApproved {
		return ErrInvalidState
	}
	seen := map[string]bool{}
	for i := range r.Points {
		r.Points[i].PointCode = strings.TrimSpace(r.Points[i].PointCode)
		r.Points[i].StageZone = strings.TrimSpace(r.Points[i].StageZone)
		r.Points[i].ComponentName = strings.TrimSpace(r.Points[i].ComponentName)
		p := r.Points[i]
		if p.PointCode == "" || p.RatedLoadKg <= 0 || p.ActualLoadKg < 0 || p.ClearanceMm < 0 || p.CueStart < 0 || p.CueEnd <= p.CueStart {
			return ErrInvalidInput
		}
		if seen[p.PointCode] {
			return ErrDuplicatePoint
		}
		seen[p.PointCode] = true
	}
	r.ContentDigest = DigestPoints(r.Points)
	r.RevisionNumber = len(c.Revisions) + 1
	r.CaseID = c.ID
	for i := range r.Points {
		if r.Points[i].ID == "" {
			r.Points[i].ID = fmt.Sprintf("%s-point-%d", r.ID, i+1)
		}
		r.Points[i].RevisionID = r.ID
	}
	c.Revisions = append(c.Revisions, r)
	c.CurrentRevisionID = r.ID
	c.LastAssessmentDigest = ""
	c.LastAssessment = nil
	c.Status = StatusPlanning
	c.touch("RevisionSubmitted", actor)
	return nil
}
func (c *RiggingCase) SetValidation(ok bool, digest, actor string) error {
	if c.Status == StatusFrozen {
		return ErrFrozen
	}
	c.LastAssessmentDigest = digest
	if ok {
		c.Status = StatusUnderReview
	} else {
		c.Status = StatusValidationFailed
	}
	c.touch("Validation", actor)
	return nil
}
func (c *RiggingCase) AddFinding(f SafetyFinding) error {
	if c.Status == StatusFrozen {
		return ErrFrozen
	}
	f.CaseID = c.ID
	f.RevisionID = c.CurrentRevisionID
	c.Findings = append(c.Findings, f)
	return nil
}
func (c *RiggingCase) ReplaceCurrentFindings(findings []SafetyFinding) {
	current := c.CurrentRevisionID
	for i := range c.Findings {
		if c.Findings[i].RevisionID != current {
			continue
		}
		c.Findings[i].Status = FindingResolved
	}
	for _, f := range findings {
		found := false
		for i := range c.Findings {
			if c.Findings[i].ID == f.ID {
				old := c.Findings[i]
				if old.RemediationRevisionID != "" {
					f.RemediationRevisionID = old.RemediationRevisionID
					f.RemediationNote = old.RemediationNote
				}
				c.Findings[i] = f
				found = true
				break
			}
		}
		if !found {
			c.Findings = append(c.Findings, f)
		}
	}
}
func (c *RiggingCase) AssociateRemediation(id, revisionID, note string) error {
	if c.Status == StatusFrozen {
		return ErrFrozen
	}
	if strings.TrimSpace(note) == "" || revisionID == "" || revisionID != c.CurrentRevisionID {
		return ErrInvalidInput
	}
	if c.CurrentRevision() == nil {
		return ErrInvalidState
	}
	for i := range c.Findings {
		if c.Findings[i].ID == id {
			if c.Findings[i].RevisionID == c.CurrentRevisionID {
				return ErrInvalidInput
			}
			c.Findings[i].RemediationRevisionID = revisionID
			c.Findings[i].RemediationNote = strings.TrimSpace(note)
			return nil
		}
	}
	return ErrNotFound
}
func (c *RiggingCase) ResolveFinding(id, actor, note string) error {
	if c.Status == StatusFrozen {
		return ErrFrozen
	}
	actor, note = strings.TrimSpace(actor), strings.TrimSpace(note)
	if actor == "" || note == "" {
		return ErrInvalidInput
	}
	for i := range c.Findings {
		if c.Findings[i].ID == id {
			now := time.Now().UTC()
			if c.Findings[i].RemediationRevisionID != c.CurrentRevisionID || strings.TrimSpace(c.Findings[i].RemediationNote) == "" {
				return ErrInvalidInput
			}
			c.Findings[i].Status = FindingResolved
			c.Findings[i].VerifiedBy = actor
			c.Findings[i].VerifiedAt = &now
			c.Findings[i].RemediationNote = note
			c.touch("FindingResolved", actor)
			return nil
		}
	}
	return ErrNotFound
}
func (c *RiggingCase) ReopenFinding(id, actor string) error {
	if c.Status == StatusFrozen {
		return ErrFrozen
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return ErrInvalidInput
	}
	for i := range c.Findings {
		if c.Findings[i].ID == id {
			if c.Findings[i].RevisionID != c.CurrentRevisionID {
				return ErrStaleAssessment
			}
			c.Findings[i].Status = FindingOpen
			c.Findings[i].VerifiedBy = ""
			c.Findings[i].VerifiedAt = nil
			c.touch("FindingReopened", actor)
			return nil
		}
	}
	return ErrNotFound
}
func (c *RiggingCase) OpenBlocking() bool {
	for _, f := range c.Findings {
		if f.Severity == SeverityBlocking && f.Status != FindingResolved {
			return true
		}
	}
	return false
}
