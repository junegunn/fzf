// +build windows

package util

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// ExecCommand executes the given command with cmd
func ExecCommand(command string, setpgid bool) *exec.Cmd {
	return ExecCommandWith("cmd", command)
}

// ExecCommandWith executes the given command with cmd. _shell parameter is
// ignored on Windows.
// FIXME: setpgid is unused. We set it in the Unix implementation so that we
// can kill preview process with its child processes at once.
func ExecCommandWith(_shell string, command string, setpgid bool) *exec.Cmd {
	cmd := exec.Command("cmd")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    false,
		CmdLine:       fmt.Sprintf(` /v:on/s/c "%s"`, command),
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
