package domain

import (
	"sort"
	"time"
)

type AuditFilter struct {
	Type, Actor string
	Since       time.Time
	Limit       int
}

func (c *RiggingCase) AuditLog(filter AuditFilter) []AuditEvent {
	out := make([]AuditEvent, 0)
	for _, a := range c.Audit {
		if filter.Type != "" && a.Type != filter.Type {
			continue
		}
		if filter.Actor != "" && a.Actor != filter.Actor {
			continue
		}
		if !filter.Since.IsZero() && a.At.Before(filter.Since) {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[len(out)-filter.Limit:]
	}
	return out
}
func (c *RiggingCase) Record(message, actor string) {
	if c.Status == StatusFrozen {
		return
	}
	c.touch(message, actor)
}
func (c *RiggingCase) RevisionByID(id string) (*PlanRevision, error) {
	for i := range c.Revisions {
		if c.Revisions[i].ID == id {
			return &c.Revisions[i], nil
		}
	}
	return nil, ErrNotFound
}
func (c *RiggingCase) FindingByID(id string) (*SafetyFinding, error) {
	for i := range c.Findings {
		if c.Findings[i].ID == id {
			return &c.Findings[i], nil
		}
	}
	return nil, ErrNotFound
}
func (c *RiggingCase) ReviewDecisionsFor(id string) []ReviewDecision {
	out := []ReviewDecision{}
	for _, r := range c.Reviews {
		if r.RevisionID == id {
			out = append(out, r)
		}
	}
	return out
}
