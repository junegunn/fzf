//go:build windows

package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"syscall"
)

type shellType int

const (
	shellTypeUnknown shellType = iota
	shellTypeCmd
	shellTypePowerShell
)

var escapeRegex = regexp.MustCompile(`[&|<>()^%!"]`)

type Executor struct {
	shell     string
	shellType shellType
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

	shellType := shellTypeUnknown
	basename := filepath.Base(shell)
	if len(args) > 0 {
		args = args[1:]
	} else if strings.HasPrefix(basename, "cmd") {
		shellType = shellTypeCmd
		args = []string{"/s/c"}
	} else if strings.HasPrefix(basename, "pwsh") || strings.HasPrefix(basename, "powershell") {
		shellType = shellTypePowerShell
		args = []string{"-NoProfile", "-Command"}
	} else {
		args = []string{"-c"}
	}
	return &Executor{shell: shell, shellType: shellType, args: args}
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
	if x.shellType == shellTypeCmd {
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
		os.Exit(127)
	}
	err = cmd.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
	}
	os.Exit(0)
}

func escapeArg(s string) string {
	b := make([]byte, 0, len(s)+2)
	b = append(b, '"')
	slashes := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		default:
			slashes = 0
		case '\\':
			slashes++
		case '"':
			for ; slashes > 0; slashes-- {
				b = append(b, '\\')
			}
			b = append(b, '\\')
		}
		b = append(b, c)
	}
	for ; slashes > 0; slashes-- {
		b = append(b, '\\')
	}
	b = append(b, '"')
	return escapeRegex.ReplaceAllStringFunc(string(b), func(match string) string {
		return "^" + match
	})
}

func (x *Executor) QuoteEntry(entry string) string {
	switch x.shellType {
	case shellTypeCmd:
		/* Manually tested with the following commands:
		   fzf --preview "echo {}"
		   fzf --preview "type {}"
		   echo .git\refs\| fzf --preview "dir {}"
		   echo .git\refs\\| fzf --preview "dir {}"
		   echo .git\refs\\\| fzf --preview "dir {}"
		   reg query HKCU | fzf --reverse --bind "enter:reload(reg query {})"
		   fzf --disabled --preview "echo {q} {n} {}" --query "&|<>()@^%!"
		   fd -H --no-ignore -td -d 4 | fzf --preview "dir {}"
		   fd -H --no-ignore -td -d 4 | fzf --preview "eza {}" --preview-window up
		   fd -H --no-ignore -td -d 4 | fzf --preview "eza --color=always --tree --level=3 --icons=always {}"
		   fd -H --no-ignore -td -d 4 | fzf --preview ".\eza.exe --color=always --tree --level=3 --icons=always {}" --with-shell "powershell -NoProfile -Command"
		*/
		return escapeArg(entry)
	case shellTypePowerShell:
		escaped := strings.ReplaceAll(entry, `"`, `\"`)
		return "'" + strings.ReplaceAll(escaped, "'", "''") + "'"
	default:
		return "'" + strings.ReplaceAll(entry, "'", "'\\''") + "'"
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
