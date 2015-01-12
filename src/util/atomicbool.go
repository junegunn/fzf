package util

import "sync"

// AtomicBool is a boxed-class that provides synchronized access to the
// underlying boolean value
type AtomicBool struct {
	mutex sync.Mutex
	state bool
}

// NewAtomicBool returns a new AtomicBool
func NewAtomicBool(initialState bool) *AtomicBool {
	return &AtomicBool{
		mutex: sync.Mutex{},
		state: initialState}
}

// Get returns the current boolean value synchronously
func (a *AtomicBool) Get() bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.state
}

// Set updates the boolean value synchronously
func (a *AtomicBool) Set(newState bool) bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.state = newState
	return a.state
}
