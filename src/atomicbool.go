package fzf

import "sync"

type AtomicBool struct {
	mutex sync.Mutex
	state bool
}

func NewAtomicBool(initialState bool) *AtomicBool {
	return &AtomicBool{
		mutex: sync.Mutex{},
		state: initialState}
}

func (a *AtomicBool) Get() bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.state
}

func (a *AtomicBool) Set(newState bool) bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.state = newState
	return a.state
}
