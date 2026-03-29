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

	"golang.org/x/sys/windows"
)

type shellType int

const (
	shellTypeUnknown shellType = iota
	shellTypeCmd
	shellTypePowerShell
	shellTypePwsh
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
	} else if strings.HasPrefix(basename, "pwsh") {
		shellType = shellTypePwsh
		args = []string{"-NoProfile", "-Command"}
	} else if strings.HasPrefix(basename, "powershell") {
		shellType = shellTypePowerShell
		args = []string{"-NoProfile", "-Command"}
	} else {
		args = []string{"-c"}
	}
	return &Executor{shell: shell, shellType: shellType, args: args}
}

// ExecCommand executes the given command with $SHELL
//
// On Windows, setpgid controls whether the spawned process is placed in a new
// process group (so that it can be signaled independently, e.g. for previews).
// However, we only do this for "pwsh" and non-standard shells, because cmd.exe
// and Windows PowerShell ("powershell.exe") don't always exit on Ctrl-Break.
//
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

	var creationFlags uint32
	// Set new process group for pwsh (PowerShell 7+) and unknown/posix-ish shells
	if setpgid && (x.shellType == shellTypePwsh || x.shellType == shellTypeUnknown) {
		creationFlags = windows.CREATE_NEW_PROCESS_GROUP
	}

	var cmd *exec.Cmd
	if x.shellType == shellTypeCmd {
		cmd = exec.Command(shell)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    false,
			CmdLine:       fmt.Sprintf(`%s "%s"`, strings.Join(x.args, " "), command),
			CreationFlags: creationFlags,
		}
	} else {
		args := x.args
		if setpgid && x.shellType == shellTypePwsh {
			// pwsh needs -NonInteractive flag to exit on Ctrl-Break
			args = append([]string{"-NonInteractive"}, x.args...)
		}
		cmd = exec.Command(shell, append(args, command)...)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    false,
			CreationFlags: creationFlags,
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
	case shellTypePowerShell, shellTypePwsh:
		escaped := strings.ReplaceAll(entry, `"`, `\"`)
		return "'" + strings.ReplaceAll(escaped, "'", "''") + "'"
	default:
		return "'" + strings.ReplaceAll(entry, "'", "'\\''") + "'"
	}
}

// KillCommand kills the process for the given command
func KillCommand(cmd *exec.Cmd) error {
	// Safely handle nil command or process.
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	// If it has its own process group, we can send it Ctrl-Break
	if cmd.SysProcAttr != nil && cmd.SysProcAttr.CreationFlags&windows.CREATE_NEW_PROCESS_GROUP != 0 {
		if err := windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(cmd.Process.Pid)); err == nil {
			return nil
		}
	}
	// If it's the same process group, or if sending the console control event
	// fails (e.g., no console, different console, or process already exited),
	// fall back to a standard kill.  This probably won't *help* if there's I/O
	// going on, because Wait() will still hang until the I/O finishes unless we
	// hard-kill the entire process group.  But it doesn't hurt to try!
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
