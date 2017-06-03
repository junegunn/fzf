// +build windows

package util

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/mattn/go-shellwords"
)

// ExecCommand executes the given command with $SHELL
func ExecCommand(command string) *exec.Cmd {
	args, _ := shellwords.Parse(command)
	allArgs := make([]string, len(args)+1)
	allArgs[0] = "/c"
	copy(allArgs[1:], args)
	return exec.Command("cmd", allArgs...)
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
