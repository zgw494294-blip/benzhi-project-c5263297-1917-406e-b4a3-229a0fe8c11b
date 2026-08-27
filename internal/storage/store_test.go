package storage

import (
	"stage-rigging-clearance/internal/domain"
	"testing"
	"time"
)

func TestStore(t *testing.T) {
	s, _ := New("")
	c := domain.NewCase("1", "演出", "区", "人", time.Now())
	if s.Save(c, 1) != nil {
		t.Fatal()
	}
	if _, e := s.Get("1"); e != nil {
		t.Fatal(e)
	}
}
