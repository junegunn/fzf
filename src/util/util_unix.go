// +build !windows

package util

import (
	"os"
	"os/exec"
	"syscall"
)

// ExecCommand executes the given command with $SHELL
func ExecCommand(command string, setpgid bool) *exec.Cmd {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	return ExecCommandWith(shell, command, setpgid)
}

// ExecCommandWith executes the given command with the specified shell
func ExecCommandWith(shell string, command string, setpgid bool) *exec.Cmd {
	cmd := exec.Command(shell, "-c", command)
	if setpgid {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	return cmd
}

// KillCommand kills the process for the given command
func KillCommand(cmd *exec.Cmd) error {
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

// IsWindows returns true on Windows
func IsWindows() bool {
	return false
}

// SetNonblock executes syscall.SetNonblock on file descriptor
func SetNonblock(file *os.File, nonblock bool) {
	syscall.SetNonblock(int(file.Fd()), nonblock)
}

// Read executes syscall.Read on file descriptor
func Read(fd int, b []byte) (int, error) {
	return syscall.Read(int(fd), b)
}
