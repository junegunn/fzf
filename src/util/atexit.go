package util

import (
	"os"
	"sync"
)

var atExitFuncs []func()

// AtExit registers the function fn to be called on program termination.
// The functions will be called in reverse order they were registered.
func AtExit(fn func()) {
	if fn == nil {
		panic("AtExit called with nil func")
	}
	once := &sync.Once{}
	atExitFuncs = append(atExitFuncs, func() {
		once.Do(fn)
	})
}

// RunAtExitFuncs runs any functions registered with AtExit().
func RunAtExitFuncs() {
	fns := atExitFuncs
	for i := len(fns) - 1; i >= 0; i-- {
		fns[i]()
	}
}

// Exit executes any functions registered with AtExit() then exits the program
// with os.Exit(code).
//
// NOTE: It must be used instead of os.Exit() since calling os.Exit() terminates
// the program before any of the AtExit functions can run.
func Exit(code int) {
	defer os.Exit(code)
	RunAtExitFuncs()
}
