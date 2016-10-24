// +build !windows

package fzf

import (
	"os"
	"os/signal"
	"syscall"
)

func notifyOnResize(resizeChan chan<- os.Signal) {
	signal.Notify(resizeChan, syscall.SIGWINCH)
}
