package sg

import (
	"iter"
	"sync"
)

// Simple set implementation.
type Set[T comparable] struct {
	sync.RWMutex
	s map[T]struct{}
}

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{s: make(map[T]struct{})}
}

func (s *Set[T]) AddAll(k ...T) {
	s.Lock()
	defer s.Unlock()
	for _, v := range k {
		s.s[v] = struct{}{}
	}
}

func (s *Set[T]) Add(k T) {
	s.Lock()
	defer s.Unlock()
	s.s[k] = struct{}{}
}

func (s *Set[T]) Remove(k T) bool {
	s.Lock()
	defer s.Unlock()
	_, ok := s.s[k]
	delete(s.s, k)
	return ok
}

func (s *Set[T]) Has(k T) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.s[k]
	return ok
}

func (s *Set[T]) Size() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.s)
}

func (s *Set[T]) Elements() iter.Seq[T] {
	return func(yield func(T) bool) {
		s.RLock()
		defer s.RUnlock()
		for k := range s.s {
			yield(k)
		}
	}
}

func (s *Set[T]) Slice() []T {
	s.RLock()
	defer s.RUnlock()
	keys := make([]T, 0, len(s.s))
	for k := range s.s {
		keys = append(keys, k)
	}
	return keys
}
