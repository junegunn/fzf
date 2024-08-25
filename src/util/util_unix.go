//go:build !windows

package util

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

type Executor struct {
	shell   string
	args    []string
	escaper *strings.Replacer
}

func NewExecutor(withShell string) *Executor {
	shell := os.Getenv("SHELL")
	args := strings.Fields(withShell)
	if len(args) > 0 {
		shell = args[0]
		args = args[1:]
	} else {
		if len(shell) == 0 {
			shell = "sh"
		}
		args = []string{"-c"}
	}

	var escaper *strings.Replacer
	tokens := strings.Split(shell, "/")
	if tokens[len(tokens)-1] == "fish" {
		// https://fishshell.com/docs/current/language.html#quotes
		// > The only meaningful escape sequences in single quotes are \', which
		// > escapes a single quote and \\, which escapes the backslash symbol.
		escaper = strings.NewReplacer("\\", "\\\\", "'", "\\'")
	} else {
		escaper = strings.NewReplacer("'", "'\\''")
	}
	return &Executor{shell, args, escaper}
}

// ExecCommand executes the given command with $SHELL
func (x *Executor) ExecCommand(command string, setpgid bool) *exec.Cmd {
	cmd := exec.Command(x.shell, append(x.args, command)...)
	if setpgid {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	return cmd
}

func (x *Executor) QuoteEntry(entry string) string {
	return "'" + x.escaper.Replace(entry) + "'"
}

func (x *Executor) Become(stdin *os.File, environ []string, command string) {
	shellPath, err := exec.LookPath(x.shell)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fzf (become): %s\n", err.Error())
		os.Exit(127)
	}
	args := append([]string{shellPath}, append(x.args, command)...)
	SetStdin(stdin)
	syscall.Exec(shellPath, args, environ)
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
