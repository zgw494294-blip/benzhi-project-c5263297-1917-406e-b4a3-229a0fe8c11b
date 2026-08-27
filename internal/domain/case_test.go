package domain

import (
	"testing"
	"time"
)

func TestDuplicatePoint(t *testing.T) {
	c := NewCase("c", "演出", "主舞台", "甲", time.Now())
	_ = c.StartPlanning("甲")
	r := PlanRevision{ID: "r", Points: []LoadPoint{{PointCode: "A", RatedLoadKg: 1, CueStart: 1, CueEnd: 2}, {PointCode: "A", RatedLoadKg: 1, CueStart: 2, CueEnd: 3}}}
	if c.AddRevision(r, "甲") != ErrDuplicatePoint {
		t.Fatal("expected duplicate")
	}
}
