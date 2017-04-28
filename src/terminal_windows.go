// +build windows

package fzf

import (
	"os"
)

func notifyOnResize(resizeChan chan<- os.Signal) {
	// TODO
}

func notifyStop(p *os.Process) {
	// NOOP
}

func notifyOnCont(resizeChan chan<- os.Signal) {
	// NOOP
}
