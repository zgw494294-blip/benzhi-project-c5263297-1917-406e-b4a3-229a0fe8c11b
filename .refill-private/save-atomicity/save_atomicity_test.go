package saveatomicity_test

import (
	"errors"
	"os"
	"path/filepath"
	"stage-rigging-clearance/internal/domain"
	"stage-rigging-clearance/internal/storage"
	"testing"
	"time"
)

func TestSaveFailureDoesNotPublishState(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "snapshot.json"), 0755); err != nil {
		t.Fatal(err)
	}
	c := domain.NewCase("case-save-failure", "演出", "主舞台", "负责人", time.Now())
	if err := c.StartPlanning("负责人"); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(c, 1); err == nil {
		t.Fatal("expected snapshot persistence failure")
	}
	if _, err := s.Get(c.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("failed save published case: %v", err)
	}
	if got := s.EventCount(); got != 0 {
		t.Fatalf("failed save advanced event count to %d", got)
	}
	if err := os.Remove(filepath.Join(dir, "snapshot.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.New(dir); err != nil {
		t.Fatalf("failed save left unrecoverable storage: %v", err)
	}
}
