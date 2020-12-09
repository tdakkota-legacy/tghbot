package storage

import (
	"context"
	"errors"
	"sync"
)

type Storage interface {
	Add(ctx context.Context, m Mapping) error
	Remove(ctx context.Context, m Mapping) error
	Get(ctx context.Context, peer Peer) ([]Mapping, error)
	List(ctx context.Context) ([]Mapping, error)
}

var ErrNotFound = errors.New("mapping not found")

type InMemoryStorage struct {
	mappings map[Peer][]Mapping
	lock     sync.RWMutex
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		mappings: map[Peer][]Mapping{},
	}
}

func (s *InMemoryStorage) Add(ctx context.Context, mapping Mapping) error {
	s.lock.Lock()
	s.mappings[mapping.Peer] = append(s.mappings[mapping.Peer], mapping)
	s.lock.Unlock()

	return nil
}

func (s *InMemoryStorage) Remove(ctx context.Context, mapping Mapping) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	a := s.mappings[mapping.Peer]
	for i, m := range a {
		if m.Repo == mapping.Repo {
			// Remove the element at index i from a.
			a[i] = a[len(a)-1]      // Copy last element to index i.
			a[len(a)-1] = Mapping{} // Erase last element (write zero value).
			a = a[:len(a)-1]        // Truncate slice.
			s.mappings[mapping.Peer] = a
			return nil
		}
	}
	return nil
}

func (s *InMemoryStorage) Get(ctx context.Context, peer Peer) ([]Mapping, error) {
	s.lock.RLock()
	r := s.mappings[peer]
	s.lock.RUnlock()
	return r, nil
}

func (s *InMemoryStorage) List(ctx context.Context) ([]Mapping, error) {
	s.lock.RLock()
	r := make([]Mapping, 0, len(s.mappings))
	for _, m := range s.mappings {
		r = append(r, m...)
	}
	s.lock.RUnlock()

	return r, nil
}
