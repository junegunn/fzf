//go:build !windows

package fzf

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

func notifyOnResize(ctx context.Context, resizeChan chan<- os.Signal) {
	signal.Notify(resizeChan, syscall.SIGWINCH)
	go func() {
		<-ctx.Done()
		signal.Stop(resizeChan)
	}()
}

func notifyStop(p *os.Process) {
	pid := p.Pid
	pgid, err := unix.Getpgid(pid)
	if err == nil {
		pid = pgid * -1
	}
	unix.Kill(pid, syscall.SIGTSTP)
}
