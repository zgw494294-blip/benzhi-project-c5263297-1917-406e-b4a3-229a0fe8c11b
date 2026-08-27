package assessment

import (
	"fmt"
	"math"
	"sort"
	"stage-rigging-clearance/internal/domain"
	"strings"
)

const RuleVersion = "rigging-rules-v1"

func Evaluate(c *domain.RiggingCase, r *domain.PlanRevision) Result {
	out := Result{RevisionID: r.ID, InputDigest: r.ContentDigest, RuleVersion: RuleVersion, ZoneLoads: map[string]float64{}, PointMetrics: []PointMetric{}, Findings: []domain.SafetyFinding{}}
	var sx, sy, total float64
	for _, p := range r.Points {
		u := p.ActualLoadKg / p.RatedLoadKg
		out.PointMetrics = append(out.PointMetrics, PointMetric{p.PointCode, u})
		out.ZoneLoads[p.StageZone] += p.ActualLoadKg
		sx += p.PositionXmm * p.ActualLoadKg
		sy += p.PositionYmm * p.ActualLoadKg
		total += p.ActualLoadKg
		if u > 1 {
			out.Findings = append(out.Findings, finding(c, r, "OVERLOAD", domain.SeverityBlocking, fmt.Sprintf("吊点 %s 利用率 %.1f%% 超过额定载荷", p.PointCode, u*100), []string{p.PointCode}))
		}
		if p.ClearanceMm < 500 {
			out.Findings = append(out.Findings, finding(c, r, "CLEARANCE", domain.SeverityBlocking, fmt.Sprintf("吊点 %s 净空 %.0fmm 小于 500mm", p.PointCode, p.ClearanceMm), []string{p.PointCode}))
		}
	}
	out.ZoneMetrics = ZoneMetrics(r.Points)
	if total > 0 {
		cx, cy := sx/total, sy/total
		out.Eccentricity = math.Sqrt(cx*cx + cy*cy)
		if out.Eccentricity > 1000 {
			out.Findings = append(out.Findings, finding(c, r, "ECCENTRICITY", domain.SeverityWarning, fmt.Sprintf("整体偏心量 %.0fmm", out.Eccentricity), nil))
		}
	}
	for i := 0; i < len(r.Points); i++ {
		for j := i + 1; j < len(r.Points); j++ {
			a, b := r.Points[i], r.Points[j]
			if a.StageZone == b.StageZone && a.CueStart < b.CueEnd && b.CueStart < a.CueEnd && distance(a, b) < 1000 {
				out.Findings = append(out.Findings, finding(c, r, "SPACE_CONFLICT", domain.SeverityBlocking, fmt.Sprintf("构件 %s 与 %s 在同一时段空间冲突", a.PointCode, b.PointCode), []string{a.PointCode, b.PointCode}))
			}
		}
	}
	sort.Slice(out.Findings, func(i, j int) bool {
		if out.Findings[i].RuleCode != out.Findings[j].RuleCode {
			return out.Findings[i].RuleCode < out.Findings[j].RuleCode
		}
		return out.Findings[i].ID < out.Findings[j].ID
	})
	return out
}
func distance(a, b domain.LoadPoint) float64 {
	x := a.PositionXmm - b.PositionXmm
	y := a.PositionYmm - b.PositionYmm
	return math.Sqrt(x*x + y*y)
}
func finding(c *domain.RiggingCase, r *domain.PlanRevision, rule string, s domain.Severity, msg string, ids []string) domain.SafetyFinding {
	ids = append([]string(nil), ids...)
	sort.Strings(ids)
	return domain.SafetyFinding{ID: fmt.Sprintf("finding-%s-%s-%s", r.ID, rule, strings.Join(ids, "+")), CaseID: c.ID, RevisionID: r.ID, RuleCode: rule, Severity: s, Message: msg, RelatedPointIDs: ids, Status: domain.FindingOpen}
}
