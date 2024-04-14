//go:build !windows

package util

import (
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

// ExecCommand executes the given command with $SHELL $FZF_SHELL_FLAG 
func ExecCommand(command string, setpgid bool) *exec.Cmd {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	shellFlag := os.Getenv("FZF_SHELL_FLAG")
	if len(shellFlag) == 0 {
		shellFlag = "-c"
	}
	return ExecCommandWith(shell, shellFlag, command, setpgid)
}

// ExecCommandWith executes the given command with the specified shell and flag
func ExecCommandWith(shell string, shellFlag string, command string, setpgid bool) *exec.Cmd {
	cmd := exec.Command(shell, shellFlag, command)
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

func SetStdin(file *os.File) {
	unix.Dup2(int(file.Fd()), 0)
}
