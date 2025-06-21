package util

import "sync"

// ConcurrentSet is a thread-safe set implementation.
type ConcurrentSet[T comparable] struct {
	lock  sync.RWMutex
	items map[T]struct{}
}

// NewConcurrentSet creates a new ConcurrentSet.
func NewConcurrentSet[T comparable]() *ConcurrentSet[T] {
	return &ConcurrentSet[T]{
		items: make(map[T]struct{}),
	}
}

// Add adds an item to the set.
func (s *ConcurrentSet[T]) Add(item T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.items[item] = struct{}{}
}

// Remove removes an item from the set.
func (s *ConcurrentSet[T]) Remove(item T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.items, item)
}

// ForEach iterates over each item in the set and applies the provided function.
func (s *ConcurrentSet[T]) ForEach(fn func(item T)) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for item := range s.items {
		fn(item)
	}
}
