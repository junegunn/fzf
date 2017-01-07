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
