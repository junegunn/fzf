//go:build !windows

package fzf

/*
#include <signal.h>
#include <unistd.h>
#include <sys/types.h>

// Wrapper for kill(2)
int native_kill(pid_t pid, int sig) {
    return kill(pid, sig);
}

// Wrapper for getpgid(2)
pid_t native_getpgid(pid_t pid) {
    return getpgid(pid);
}
*/
import "C"
import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"fmt"

//	"golang.org/x/sys/unix"
)

// Kill sends a signal to a process.
func Kill(pid int, signal syscall.Signal) error {
	// C.kill returns 0 on success, -1 on failure
	res := C.native_kill(C.pid_t(pid), C.int(signal))
	if res != 0 {
		return fmt.Errorf("kill failed for pid %d", pid)
	}
	return nil
}


// Getpgid returns the process group ID for the given process ID.
func Getpgid(pid int) (int, error) {
	// C.getpgid returns the PGID or -1 on failure
	res := C.native_getpgid(C.pid_t(pid))
	if res < 0 {
		return 0, fmt.Errorf("getpgid failed for pid %d", pid)
	}
	return int(res), nil
}



func notifyOnResize(resizeChan chan<- os.Signal) {
	signal.Notify(resizeChan, syscall.SIGWINCH)
}

func notifyStop(p *os.Process) {
	pid := p.Pid
	pgid, err := Getpgid(pid)
	if err == nil {
		pid = pgid * -1
	}
	Kill(pid, syscall.SIGSTOP)
}

func notifyOnCont(resizeChan chan<- os.Signal) {
	signal.Notify(resizeChan, syscall.SIGCONT)
}

func quoteEntry(entry string) string {
	return "'" + strings.Replace(entry, "'", "'\\''", -1) + "'"
}
