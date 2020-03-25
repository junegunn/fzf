package util

import (
	"sync/atomic"
)

// AtomicBool is a boxed-class that provides synchronized access to the
// underlying boolean value
type AtomicBool struct {
	state int32 // "1" is true, "0" is false
}

// NewAtomicBool returns a new AtomicBool
func NewAtomicBool(initialState bool) *AtomicBool {
	var state int32 = 0
	if initialState == true {
		state = 1
	}
	return &AtomicBool{state: state}
}

// Get returns the current boolean value synchronously
func (a *AtomicBool) Get() bool {
	if atomic.LoadInt32(&a.state) != 0 {
		return true
	}
	return false
}

// Set updates the boolean value synchronously
func (a *AtomicBool) Set(newState bool) bool {
	var state int32 = 0
	if newState == true {
		state = 1
	}
	atomic.StoreInt32(&a.state, state)
	return newState
}
