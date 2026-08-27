package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

func SnapshotDigest(c *RiggingCase, r *PlanRevision) string {
	snap, _ := json.Marshal(struct {
		CaseID, PerformanceName, StageZones, OwnerName, Date string
		Revision                                             PlanRevision
	}{c.ID, c.PerformanceName, c.StageZones, c.OwnerName, c.PerformanceDate.Format("2006-01-02"), *r})
	h := sha256.Sum256(snap)
	return hex.EncodeToString(h[:])
}

func (c *RiggingCase) Freeze(issuedBy string) (ClearancePermit, error) {
	if strings.TrimSpace(issuedBy) == "" {
		return ClearancePermit{}, ErrInvalidInput
	}
	issuedBy = strings.TrimSpace(issuedBy)
	if e := c.CanFreeze(); e != nil {
		return ClearancePermit{}, e
	}
	r := c.CurrentRevision()
	if r == nil {
		return ClearancePermit{}, ErrInvalidState
	}
	sd := SnapshotDigest(c, r)
	code := "CLR-" + sd[:12]
	vh := sha256.Sum256([]byte(code + sd))
	p := ClearancePermit{ID: fmtID("permit", 1), PermitCode: code, CaseID: c.ID, FrozenRevisionID: r.ID, SnapshotDigest: sd, IssuedBy: issuedBy, IssuedAt: time.Now().UTC(), VerificationDigest: hex.EncodeToString(vh[:])}
	c.Permit = &p
	c.Status = StatusFrozen
	c.touch("Frozen", issuedBy)
	return p, nil
}
