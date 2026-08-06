package observation

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SourceKind identifies supported observed source kinds.
type SourceKind string

const (
	SourceKindSecret    SourceKind = "Secret"
	SourceKindConfigMap SourceKind = "ConfigMap"
)

// SourceIdentity is the durable key for observed objects.
type SourceIdentity struct {
	APIVersion string
	Kind       SourceKind
	Namespace  string
	Name       string
}

// Record is the durable observation model used by the spike prototype.
type Record struct {
	Identity                    SourceIdentity
	LastSeenResourceVersion     string
	LastRelevantResourceVersion string
	LastRelevantChangeTime      time.Time
	RelevantFingerprint         string
}

// Store isolates the durable model from observer logic.
type Store interface {
	Get(context.Context, SourceIdentity) (Record, bool, error)
	Upsert(context.Context, Record) error
}

// MemoryStore is a deterministic prototype store used for tests and spikes.
type MemoryStore struct {
	mu      sync.RWMutex
	records map[string]Record
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{records: map[string]Record{}}
}

func (s *MemoryStore) Get(_ context.Context, identity SourceIdentity) (Record, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[key(identity)]
	return record, ok, nil
}

func (s *MemoryStore) Upsert(_ context.Context, record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[key(record.Identity)] = record
	return nil
}

func key(identity SourceIdentity) string {
	return fmt.Sprintf("%s|%s|%s|%s", identity.APIVersion, identity.Kind, identity.Namespace, identity.Name)
}
