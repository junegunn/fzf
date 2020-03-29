package util

import (
	"sync/atomic"
)

func convertBoolToInt32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// AtomicBool is a boxed-class that provides synchronized access to the
// underlying boolean value
type AtomicBool struct {
	state int32 // "1" is true, "0" is false
}

// NewAtomicBool returns a new AtomicBool
func NewAtomicBool(initialState bool) *AtomicBool {
	return &AtomicBool{state: convertBoolToInt32(initialState)}
}

// Get returns the current boolean value synchronously
func (a *AtomicBool) Get() bool {
	return atomic.LoadInt32(&a.state) == 1
}

// Set updates the boolean value synchronously
func (a *AtomicBool) Set(newState bool) bool {
	atomic.StoreInt32(&a.state, convertBoolToInt32(newState))
	return newState
}
