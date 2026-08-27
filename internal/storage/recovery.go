package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"stage-rigging-clearance/internal/domain"
)

func (s *Store) ExportCase(id, path string) error {
	c, e := s.Get(id)
	if e != nil {
		return e
	}
	b, e := json.MarshalIndent(c, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(path, b, 0644)
}
func ImportCase(path string) (*domain.RiggingCase, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	var c domain.RiggingCase
	if e = json.Unmarshal(b, &c); e != nil {
		return nil, e
	}
	if c.ID == "" {
		return nil, domain.ErrInvalidInput
	}
	return &c, nil
}
func (s *Store) ReplaceSnapshot(snapshot Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if snapshot.Cases == nil || snapshot.Permits == nil {
		return domain.ErrInvalidInput
	}
	s.cases = snapshot.Cases
	s.permits = snapshot.Permits
	s.seq = snapshot.Version
	return s.persist()
}
func (s *Store) SnapshotPath() string {
	if s.dir == "" {
		return ""
	}
	return filepath.Join(s.dir, "snapshot.json")
}
func (s *Store) EventCount() int { s.mu.RLock(); defer s.mu.RUnlock(); return s.seq }
