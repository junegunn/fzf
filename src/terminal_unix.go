//go:build !windows

package fzf

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func notifyOnResize(resizeChan chan<- os.Signal) {
	signal.Notify(resizeChan, syscall.SIGWINCH)
}

func notifyStop(p *os.Process) {
	pid := p.Pid
	pgid, err := unix.Getpgid(pid)
	if err == nil {
		pid = pgid * -1
	}
	unix.Kill(pid, syscall.SIGSTOP)
}

func notifyOnCont(resizeChan chan<- os.Signal) {
	signal.Notify(resizeChan, syscall.SIGCONT)
}

func quoteEntry(entry string) string {
	return "'" + strings.Replace(entry, "'", "'\\''", -1) + "'"
}
