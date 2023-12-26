//go:build !windows

package fzf

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

var escaper *strings.Replacer

func init() {
	tokens := strings.Split(os.Getenv("SHELL"), "/")
	if tokens[len(tokens)-1] == "fish" {
		// https://fishshell.com/docs/current/language.html#quotes
		// > The only meaningful escape sequences in single quotes are \', which
		// > escapes a single quote and \\, which escapes the backslash symbol.
		escaper = strings.NewReplacer("\\", "\\\\", "'", "\\'")
	} else {
		escaper = strings.NewReplacer("'", "'\\''")
	}
}

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
	return "'" + escaper.Replace(entry) + "'"
}
