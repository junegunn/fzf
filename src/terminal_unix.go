//go:build !windows

package fzf

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func notifyOnResize(resizeChan chan<- os.Signal) {
	signal.Notify(resizeChan, syscall.SIGWINCH)
}

func notifyStop(p *os.Process) {
	p.Signal(syscall.SIGSTOP)
}

func notifyOnCont(resizeChan chan<- os.Signal) {
	signal.Notify(resizeChan, syscall.SIGCONT)
}

func quoteEntry(entry string) string {
	return "'" + strings.Replace(entry, "'", "'\\''", -1) + "'"
}
