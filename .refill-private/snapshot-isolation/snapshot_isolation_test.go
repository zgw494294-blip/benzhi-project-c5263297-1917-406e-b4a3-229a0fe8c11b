package snapshotisolation_test

import (
	"errors"
	"stage-rigging-clearance/internal/domain"
	"stage-rigging-clearance/internal/storage"
	"testing"
	"time"
)

func TestSnapshotMutationDoesNotPolluteStore(t *testing.T) {
	s, err := storage.New("")
	if err != nil {
		t.Fatal(err)
	}
	c := domain.NewCase("case-snapshot", "演出", "主舞台", "负责人", time.Now())
	if err := c.StartPlanning("负责人"); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(c, 1); err != nil {
		t.Fatal(err)
	}
	snapshot := s.Snapshot()
	snapshot.Cases[c.ID].PerformanceName = "外部篡改"
	snapshot.Cases["ghost"] = domain.NewCase("ghost", "幽灵演出", "主舞台", "负责人", time.Now())
	got, err := s.Get(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PerformanceName == "外部篡改" {
		t.Fatal("snapshot mutation polluted stored case")
	}
	if _, err := s.Get("ghost"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("snapshot mutation published a ghost case: %v", err)
	}
}
