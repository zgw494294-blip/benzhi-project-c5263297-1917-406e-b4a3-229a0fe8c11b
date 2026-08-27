package assessment

import (
	"math"
	"sort"
	"stage-rigging-clearance/internal/domain"
)

const (
	MinClearanceMm     = 500
	MaxEccentricityMm  = 1000
	ConflictDistanceMm = 1000
)

type ZoneMetric struct {
	Zone       string  `json:"zone"`
	LoadKg     float64 `json:"loadKg"`
	PointCount int     `json:"pointCount"`
}

func Utilization(p domain.LoadPoint) float64 {
	if p.RatedLoadKg <= 0 {
		return math.Inf(1)
	}
	return p.ActualLoadKg / p.RatedLoadKg
}
func ZoneMetrics(points []domain.LoadPoint) []ZoneMetric {
	m := map[string]*ZoneMetric{}
	for _, p := range points {
		x := m[p.StageZone]
		if x == nil {
			x = &ZoneMetric{Zone: p.StageZone}
			m[p.StageZone] = x
		}
		x.LoadKg += p.ActualLoadKg
		x.PointCount++
	}
	out := make([]ZoneMetric, 0, len(m))
	for _, x := range m {
		out = append(out, *x)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Zone < out[j].Zone })
	return out
}
func Eccentricity(points []domain.LoadPoint) float64 {
	var sx, sy, total float64
	for _, p := range points {
		sx += p.PositionXmm * p.ActualLoadKg
		sy += p.PositionYmm * p.ActualLoadKg
		total += p.ActualLoadKg
	}
	if total == 0 {
		return 0
	}
	return math.Hypot(sx/total, sy/total)
}
func TemporalOverlap(a, b domain.LoadPoint) bool {
	return a.CueStart < b.CueEnd && b.CueStart < a.CueEnd
}
func SpatialConflict(a, b domain.LoadPoint) bool {
	return a.StageZone == b.StageZone && TemporalOverlap(a, b) && math.Hypot(a.PositionXmm-b.PositionXmm, a.PositionYmm-b.PositionYmm) < ConflictDistanceMm
}
func RuleDescription(code string) string {
	switch code {
	case "OVERLOAD":
		return "吊点实际载荷不得超过额定载荷"
	case "CLEARANCE":
		return "吊点净空必须达到安全阈值"
	case "SPACE_CONFLICT":
		return "同区域同场景区间内构件不得占用重叠空间"
	case "ECCENTRICITY":
		return "整体载荷偏心量应保持在建议阈值内"
	}
	return "未知核验规则"
}
