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
// truncateEvents truncates events.jsonl to the first n bytes, used to undo an
// appended event when a subsequent snapshot write fails.
func (s *Store) truncateEvents(n int64) error {
	if s.dir == "" {
		return nil
	}
	p := filepath.Join(s.dir, "events.jsonl")
	if n == 0 {
		return os.Remove(p)
	}
	return os.Truncate(p, n)
}
func (s *Store) eventLogSize() (int64, error) {
	if s.dir == "" {
		return 0, nil
	}
	p := filepath.Join(s.dir, "events.jsonl")
	fi, e := os.Stat(p)
	if os.IsNotExist(e) {
		return 0, nil
	}
	if e != nil {
		return 0, e
	}
	return fi.Size(), nil
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
	// Capture the committed state to persist before mutating the live maps,
	// so a failure in either durable step leaves the in-memory view unchanged.
	newCases := make(map[string]*domain.RiggingCase, len(s.cases)+1)
	for k, v := range s.cases {
		newCases[k] = v
	}
	newCases[c.ID] = &cp
	newPermits := s.permits
	if cp.Permit != nil {
		newPermits = make(map[string]string, len(s.permits)+1)
		for k, v := range s.permits {
			newPermits[k] = v
		}
		newPermits[cp.Permit.PermitCode] = cp.ID
	}
	newSeq := s.seq + 1
	event := Event{Seq: newSeq, CaseID: c.ID, Type: "snapshot", At: cp.UpdatedAt, Digest: SnapshotDigest(&cp), PrevDigest: s.lastDigest}
	// Stage the prospective state, then make the durable snapshot replace first.
	// If the (atomic temp+rename) snapshot write fails, snapshot.json is still
	// the previous one, so nothing durable changed; just restore the live view.
	prevCases, prevPermits, prevSeq, prevDigest := s.cases, s.permits, s.seq, s.lastDigest
	s.cases, s.permits, s.seq = newCases, newPermits, newSeq
	if err := s.persist(); err != nil {
		s.cases, s.permits, s.seq, s.lastDigest = prevCases, prevPermits, prevSeq, prevDigest
		return err
	}
	// Snapshot is durable at newSeq; append the matching event. If this fails,
	// roll the snapshot back so the on-disk version and event log stay aligned.
	before, _ := s.eventLogSize()
	if err := s.AppendEvent(event); err != nil {
		s.cases, s.permits, s.seq, s.lastDigest = prevCases, prevPermits, prevSeq, prevDigest
		_ = s.persist()
		return err
	}
	// If the append reported success but the file did not grow, the event did
	// not actually land; treat it as a persistence failure and roll back.
	if s.dir != "" {
		if after, _ := s.eventLogSize(); after <= before {
			s.cases, s.permits, s.seq, s.lastDigest = prevCases, prevPermits, prevSeq, prevDigest
			_ = s.persist()
			_ = s.truncateEvents(before)
			return domain.ErrInvalidInput
		}
	}
	s.lastDigest = event.Digest
	return nil
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
