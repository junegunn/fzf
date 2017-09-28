// +build !windows

package util

import (
	"os"
	"os/exec"
	"syscall"
)

// ExecCommand executes the given command with $SHELL
func ExecCommand(command string) *exec.Cmd {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	return ExecCommandWith(shell, command)
}

// ExecCommandWith executes the given command with the specified shell
func ExecCommandWith(shell string, command string) *exec.Cmd {
	return exec.Command(shell, "-c", command)
}

// IsWindows returns true on Windows
func IsWindows() bool {
	return false
}

// SetNonBlock executes syscall.SetNonblock on file descriptor
func SetNonblock(file *os.File, nonblock bool) {
	syscall.SetNonblock(int(file.Fd()), nonblock)
}

// Read executes syscall.Read on file descriptor
func Read(fd int, b []byte) (int, error) {
	return syscall.Read(int(fd), b)
}
