package application

import (
	"stage-rigging-clearance/internal/domain"
	"stage-rigging-clearance/internal/storage"
	"testing"
)

func TestCreateRevision(t *testing.T) {
	s := New(storageMust())
	c, e := s.Create("剧目", "主舞台", "张", "2026-08-27")
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.SubmitRevision(c.ID, c.ExpectedVersion, "初版", "张", []domain.LoadPoint{{PointCode: "A", RatedLoadKg: 100, ActualLoadKg: 50, ClearanceMm: 600, CueStart: 1, CueEnd: 2}})
	if e != nil {
		t.Fatal(e)
	}
}
func storageMust() *storage.Store { s, _ := storage.New(""); return s }
