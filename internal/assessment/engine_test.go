package assessment

import (
	"stage-rigging-clearance/internal/domain"
	"testing"
	"time"
)

func TestOverload(t *testing.T) {
	c := domain.NewCase("c", "演出", "区", "人", time.Now())
	r := domain.PlanRevision{ID: "r", ContentDigest: "x", Points: []domain.LoadPoint{{PointCode: "A", RatedLoadKg: 100, ActualLoadKg: 120, ClearanceMm: 600}}}
	x := Evaluate(c, &r)
	if !x.HasBlocking() {
		t.Fatal("expected blocking")
	}
}
