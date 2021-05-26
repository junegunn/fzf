// +build windows

package util

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// Get shell to execute from SHELL environment variable, defaulting to cmd
func getShell() string {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "cmd"
	}
	return shell
}

// Check if shell is cmd or cmd.exe or similar, else assume it's Powershell
func isShellCmd(shell string) bool {
	return strings.Contains(shell, "cmd")
}

// QuoteShellEntry quotes a string appropriately for the shell
func QuoteShellEntry(entry string) string {
	if isShellCmd(getShell()) {
		return QuoteShellEntryCmd(entry)
	} else {
		return QuoteShellEntryPs(entry)
	}
}

// ExecCommand executes the given command with $SHELL
func ExecCommand(command string, setpgid bool) *exec.Cmd {
	return ExecCommandWith(getShell(), command, setpgid)
}

// ExecCommandWith executes the given command with the specified shell.
// Depending on isShellCmd(shell) this creates a command for use with cmd
// or else PS. In the latter case the command string is passed
// as a ScriptBlock which gets executed, i.e. &{<command>}.
// FIXME: setpgid is unused. We set it in the Unix implementation so that we
// can kill preview process with its child processes at once.
func ExecCommandWith(shell string, command string, setpgid bool) *exec.Cmd {
	var cmdline string
	if isShellCmd(shell) {
		cmdline = fmt.Sprintf(` /v:on/s/c "%s"`, command)
	} else {
		cmdline = fmt.Sprintf(` -NoProfile -Command "&{%s}"`, command)
	}
	cmd := exec.Command(shell)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    false,
		CmdLine:       cmdline,
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
