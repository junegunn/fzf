//go:build windows

package util

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"syscall"
)

var shellPath atomic.Value

// ExecCommand executes the given command with $SHELL
func ExecCommand(command string, setpgid bool) *exec.Cmd {
	var shell string
	if cached := shellPath.Load(); cached != nil {
		shell = cached.(string)
	} else {
		shell = os.Getenv("SHELL")
		if len(shell) == 0 {
			shell = "cmd"
		} else if strings.Contains(shell, "/") {
			out, err := exec.Command("cygpath", "-w", shell).Output()
			if err == nil {
				shell = strings.Trim(string(out), "\n")
			}
		}
		shellPath.Store(shell)
	}
	return ExecCommandWith(shell, command, setpgid)
}

// ExecCommandWith executes the given command with the specified shell
// FIXME: setpgid is unused. We set it in the Unix implementation so that we
// can kill preview process with its child processes at once.
// NOTE: For "powershell", we should ideally set output encoding to UTF8,
// but it is left as is now because no adverse effect has been observed.
func ExecCommandWith(shell string, command string, setpgid bool) *exec.Cmd {
	var cmd *exec.Cmd
	if strings.Contains(shell, "cmd") {
		cmd = exec.Command(shell)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    false,
			CmdLine:       fmt.Sprintf(` /v:on/s/c "%s"`, command),
			CreationFlags: 0,
		}
		return cmd
	}

	if strings.Contains(shell, "pwsh") || strings.Contains(shell, "powershell") {
		cmd = exec.Command(shell, "-NoProfile", "-Command", command)
	} else {
		cmd = exec.Command(shell, "-c", command)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    false,
		CreationFlags: 0,
	}
	return cmd
}

// KillCommand kills the process for the given command
func KillCommand(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}

// IsWindows returns true on Windows
func IsWindows() bool {
	return true
}

// SetNonblock executes syscall.SetNonblock on file descriptor
func SetNonblock(file *os.File, nonblock bool) {
	syscall.SetNonblock(syscall.Handle(file.Fd()), nonblock)
}

// Read executes syscall.Read on file descriptor
func Read(fd int, b []byte) (int, error) {
	return syscall.Read(syscall.Handle(fd), b)
}
