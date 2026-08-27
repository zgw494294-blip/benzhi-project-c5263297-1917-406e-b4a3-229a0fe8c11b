package application

import (
	"sort"
	"stage-rigging-clearance/internal/assessment"
	"stage-rigging-clearance/internal/domain"
)

type TimelineItem struct {
	Sequence int    `json:"sequence"`
	Type     string `json:"type"`
	Actor    string `json:"actor"`
	Message  string `json:"message"`
	At       string `json:"at"`
}
type RevisionDiff struct {
	From    string   `json:"from"`
	To      string   `json:"to"`
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
	Changed []string `json:"changed"`
}

func (s *Service) Timeline(id string) ([]TimelineItem, error) {
	c, e := s.store.Get(id)
	if e != nil {
		return nil, e
	}
	out := make([]TimelineItem, 0, len(c.Audit))
	for _, a := range c.Audit {
		out = append(out, TimelineItem{a.Seq, a.Type, a.Actor, a.Message, a.At.Format("2006-01-02 15:04:05")})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Sequence < out[j].Sequence })
	return out, nil
}
func DiffRevisions(a, b domain.PlanRevision) RevisionDiff {
	d := RevisionDiff{From: a.ID, To: b.ID}
	am := map[string]domain.LoadPoint{}
	bm := map[string]domain.LoadPoint{}
	for _, p := range a.Points {
		am[p.PointCode] = p
	}
	for _, p := range b.Points {
		bm[p.PointCode] = p
	}
	for k := range bm {
		if _, ok := am[k]; !ok {
			d.Added = append(d.Added, k)
		} else if !samePoint(am[k], bm[k]) {
			d.Changed = append(d.Changed, k)
		}
	}
	for k := range am {
		if _, ok := bm[k]; !ok {
			d.Removed = append(d.Removed, k)
		}
	}
	sort.Strings(d.Added)
	sort.Strings(d.Removed)
	sort.Strings(d.Changed)
	return d
}
func samePoint(a, b domain.LoadPoint) bool {
	return a.PointCode == b.PointCode && a.StageZone == b.StageZone && a.ComponentName == b.ComponentName && a.RatedLoadKg == b.RatedLoadKg && a.ActualLoadKg == b.ActualLoadKg && a.ClearanceMm == b.ClearanceMm && a.PositionXmm == b.PositionXmm && a.PositionYmm == b.PositionYmm && a.CueStart == b.CueStart && a.CueEnd == b.CueEnd
}
func AssessmentFresh(c *domain.RiggingCase) bool {
	r := c.CurrentRevision()
	return r != nil && c.LastAssessmentDigest == r.ContentDigest
}
func FindingChain(c *domain.RiggingCase) map[string][]domain.SafetyFinding {
	m := map[string][]domain.SafetyFinding{}
	for _, f := range c.Findings {
		m[f.RuleCode] = append(m[f.RuleCode], f)
	}
	return m
}
func ResultSummary(r assessment.Result) string { return r.RuleVersion + ":" + r.InputDigest }
