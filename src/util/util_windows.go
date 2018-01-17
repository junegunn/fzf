// +build windows

package util

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// ExecCommand executes the given command with cmd
func ExecCommand(command string) *exec.Cmd {
	return ExecCommandWith("cmd", command)
}

// ExecCommandWith executes the given command with cmd. _shell parameter is
// ignored on Windows.
func ExecCommandWith(_shell string, command string) *exec.Cmd {
	cmd := exec.Command("cmd")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    false,
		CmdLine:       fmt.Sprintf(` /v:on/s/c "%s"`, command),
		CreationFlags: 0,
	}
	return cmd
}

// IsWindows returns true on Windows
func IsWindows() bool {
	return true
}

// SetNonBlock executes syscall.SetNonblock on file descriptor
func SetNonblock(file *os.File, nonblock bool) {
	syscall.SetNonblock(syscall.Handle(file.Fd()), nonblock)
}

// Read executes syscall.Read on file descriptor
func Read(fd int, b []byte) (int, error) {
	return syscall.Read(syscall.Handle(fd), b)
}
