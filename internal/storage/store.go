package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"stage-rigging-clearance/internal/domain"
	"sync"
)

type Store struct {
	mu         sync.RWMutex
	dir        string
	cases      map[string]*domain.RiggingCase
	permits    map[string]string
	seq        int
	lastDigest string
}

func New(dir string) (*Store, error) {
	s := &Store{dir: dir, cases: map[string]*domain.RiggingCase{}, permits: map[string]string{}}
	if dir == "" {
		return s, nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	if err := s.ValidateLog(); err != nil {
		return nil, err
	}
	return s, nil
}
func (s *Store) load() error {
	b, e := os.ReadFile(filepath.Join(s.dir, "snapshot.json"))
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	var x Snapshot
	if json.Unmarshal(b, &x) != nil {
		return domain.ErrInvalidInput
	}
	s.cases = x.Cases
	s.permits = x.Permits
	if s.cases == nil {
		s.cases = map[string]*domain.RiggingCase{}
	}
	if s.permits == nil {
		s.permits = map[string]string{}
	}
	s.seq = x.Version
	return nil
}
func (s *Store) persist() error {
	if s.dir == "" {
		return nil
	}
	b, _ := json.MarshalIndent(struct {
		Cases   map[string]*domain.RiggingCase `json:"cases"`
		Permits map[string]string              `json:"permits"`
		Version int                            `json:"version"`
	}{Cases: s.cases, Permits: s.permits, Version: s.seq}, "", "  ")
	tmp := filepath.Join(s.dir, "snapshot.tmp")
	if e := os.WriteFile(tmp, b, 0644); e != nil {
		return e
	}
	return os.Rename(tmp, filepath.Join(s.dir, "snapshot.json"))
}
func (s *Store) Get(id string) (*domain.RiggingCase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.cases[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	b, _ := json.Marshal(c)
	var cp domain.RiggingCase
	_ = json.Unmarshal(b, &cp)
	return &cp, nil
}
func (s *Store) Save(c *domain.RiggingCase, expected int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.cases[c.ID]; ok && expected != old.ExpectedVersion {
		return domain.ErrVersionConflict
	}
	b, _ := json.Marshal(c)
	var cp domain.RiggingCase
	_ = json.Unmarshal(b, &cp)
	s.cases[c.ID] = &cp
	if cp.Permit != nil {
		s.permits[cp.Permit.PermitCode] = cp.ID
	}
	s.seq++
	event := Event{Seq: s.seq, CaseID: c.ID, Type: "snapshot", At: cp.UpdatedAt, Digest: SnapshotDigest(&cp), PrevDigest: s.lastDigest}
	if err := s.AppendEvent(event); err != nil {
		return err
	}
	s.lastDigest = event.Digest
	return s.persist()
}
func (s *Store) FindPermit(code string) (*domain.RiggingCase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.permits[code]
	if !ok {
		return nil, domain.ErrNotFound
	}
	c := s.cases[id]
	if c == nil || c.Permit == nil || c.Permit.PermitCode != code {
		return nil, domain.ErrNotFound
	}
	b, _ := json.Marshal(c)
	var cp domain.RiggingCase
	_ = json.Unmarshal(b, &cp)
	return &cp, nil
}
