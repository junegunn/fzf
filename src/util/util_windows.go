//go:build windows

package util

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync/atomic"
	"syscall"
)

type Executor struct {
	shell     string
	args      []string
	shellPath atomic.Value
}

func NewExecutor(withShell string) *Executor {
	shell := os.Getenv("SHELL")
	args := strings.Fields(withShell)
	if len(args) > 0 {
		shell = args[0]
	} else if len(shell) == 0 {
		shell = "cmd"
	}

	if len(args) > 0 {
		args = args[1:]
	} else if strings.Contains(shell, "cmd") {
		args = []string{"/v:on/s/c"}
	} else if strings.Contains(shell, "pwsh") || strings.Contains(shell, "powershell") {
		args = []string{"-NoProfile", "-Command"}
	} else {
		args = []string{"-c"}
	}
	return &Executor{shell: shell, args: args}
}

// ExecCommand executes the given command with $SHELL
// FIXME: setpgid is unused. We set it in the Unix implementation so that we
// can kill preview process with its child processes at once.
// NOTE: For "powershell", we should ideally set output encoding to UTF8,
// but it is left as is now because no adverse effect has been observed.
func (x *Executor) ExecCommand(command string, setpgid bool) *exec.Cmd {
	shell := x.shell
	if cached := x.shellPath.Load(); cached != nil {
		shell = cached.(string)
	} else {
		if strings.Contains(shell, "/") {
			out, err := exec.Command("cygpath", "-w", shell).Output()
			if err == nil {
				shell = strings.Trim(string(out), "\n")
			}
		}
		x.shellPath.Store(shell)
	}
	var cmd *exec.Cmd
	if strings.Contains(shell, "cmd") {
		cmd = exec.Command(shell)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    false,
			CmdLine:       fmt.Sprintf(`%s "%s"`, strings.Join(x.args, " "), command),
			CreationFlags: 0,
		}
	} else {
		cmd = exec.Command(shell, append(x.args, command)...)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    false,
			CreationFlags: 0,
		}
	}
	return cmd
}

func (x *Executor) Become(stdin *os.File, environ []string, command string) {
	cmd := x.ExecCommand(command, false)
	cmd.Stdin = stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = environ
	err := cmd.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fzf (become): %s\n", err.Error())
		Exit(127)
	}
	err = cmd.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			Exit(exitError.ExitCode())
		}
	}
	Exit(0)
}

func (x *Executor) QuoteEntry(entry string) string {
	if strings.Contains(x.shell, "cmd") {
		// backslash escaping is done here for applications
		// (see ripgrep test case in terminal_test.go#TestWindowsCommands)
		escaped := strings.Replace(entry, `\`, `\\`, -1)
		escaped = `"` + strings.Replace(escaped, `"`, `\"`, -1) + `"`
		// caret is the escape character for cmd shell
		r, _ := regexp.Compile(`[&|<>()@^%!"]`)
		return r.ReplaceAllStringFunc(escaped, func(match string) string {
			return "^" + match
		})
	} else if strings.Contains(x.shell, "pwsh") || strings.Contains(x.shell, "powershell") {
		escaped := strings.Replace(entry, `"`, `\"`, -1)
		return "'" + strings.Replace(escaped, "'", "''", -1) + "'"
	} else {
		return "'" + strings.Replace(entry, "'", "'\\''", -1) + "'"
	}
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

func SetStdin(file *os.File) {
	// No-op
}
