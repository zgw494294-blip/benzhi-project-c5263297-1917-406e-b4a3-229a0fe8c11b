package assessment

import "stage-rigging-clearance/internal/domain"

type PointMetric struct {
	PointCode   string  `json:"pointCode"`
	Utilization float64 `json:"utilization"`
}
type Result struct {
	RevisionID   string                 `json:"revisionId"`
	InputDigest  string                 `json:"inputDigest"`
	RuleVersion  string                 `json:"ruleVersion"`
	PointMetrics []PointMetric          `json:"pointMetrics"`
	ZoneLoads    map[string]float64     `json:"zoneLoads"`
	ZoneMetrics  []ZoneMetric           `json:"zoneMetrics"`
	Eccentricity float64                `json:"eccentricity"`
	Findings     []domain.SafetyFinding `json:"findings"`
}

func (r Result) HasBlocking() bool {
	for _, f := range r.Findings {
		if f.Severity == domain.SeverityBlocking {
			return true
		}
	}
	return false
}
