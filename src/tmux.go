package fzf

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/junegunn/fzf/src/tui"
	"github.com/junegunn/fzf/src/util"
)

func escapeSingleQuote(str string) string {
	return "'" + strings.ReplaceAll(str, "'", "'\\''") + "'"
}

func runTmux(args []string, opts *Options) (int, error) {
	ns := time.Now().UnixNano()

	output := filepath.Join(os.TempDir(), fmt.Sprintf("fzf-tmux-output-%d", ns))
	if err := mkfifo(output, 0666); err != nil {
		return ExitError, err
	}
	defer os.Remove(output)

	// Find fzf executable
	fzf := "fzf"
	if found, err := os.Executable(); err == nil {
		fzf = found
	}

	// Prepare arguments
	args = append([]string{"--bind=ctrl-z:ignore"}, args...)
	if opts.BorderShape == tui.BorderUndefined {
		args = append(args, "--border")
	}
	args = append(args, "--no-height")
	args = append(args, "--no-tmux")
	argStr := ""
	for _, arg := range args {
		// %q formatting escapes $'foo\nbar' to "foo\nbar"
		argStr += " " + escapeSingleQuote(arg)
	}
	argStr += ` --tmux-script "$0"`

	// Build command
	var command string
	if opts.Input == nil && util.IsTty() {
		command = fmt.Sprintf(`%q%s > %q`, fzf, argStr, output)
	} else {
		input := filepath.Join(os.TempDir(), fmt.Sprintf("fzf-tmux-input-%d", ns))
		if err := mkfifo(input, 0644); err != nil {
			return ExitError, err
		}
		defer os.Remove(input)

		go func() {
			inputFile, err := os.OpenFile(input, os.O_WRONLY, 0)
			if err != nil {
				return
			}
			if opts.Input == nil {
				io.Copy(inputFile, os.Stdin)
			} else {
				for item := range opts.Input {
					fmt.Fprint(inputFile, item+opts.PrintSep)
				}
			}
			inputFile.Close()
		}()

		command = fmt.Sprintf(`%q%s < %q > %q`, fzf, argStr, input, output)
	}

	// Get current directory
	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}

	// Set tmux options for popup placement
	// C        Both    The centre of the terminal
	// R        -x      The right side of the terminal
	// P        Both    The bottom left of the pane
	// M        Both    The mouse position
	// W        Both    The window position on the status line
	// S        -y      The line above or below the status line
	tmuxArgs := []string{"display-popup", "-E", "-B", "-d", dir}
	switch opts.Tmux.position {
	case posUp:
		tmuxArgs = append(tmuxArgs, "-xC", "-y0")
	case posDown:
		tmuxArgs = append(tmuxArgs, "-xC", "-yS")
	case posLeft:
		tmuxArgs = append(tmuxArgs, "-x0", "-yC")
	case posRight:
		tmuxArgs = append(tmuxArgs, "-xR", "-yC")
	case posCenter:
		tmuxArgs = append(tmuxArgs, "-xC", "-yC")
	}
	tmuxArgs = append(tmuxArgs, "-w"+opts.Tmux.width.String())
	tmuxArgs = append(tmuxArgs, "-h"+opts.Tmux.height.String())

	// To ensure that the options are processed by a POSIX-compliant shell,
	// we need to write the command to a temporary file and execute it with sh.
	exports := os.Environ()
	for idx, pairStr := range exports {
		pair := strings.SplitN(pairStr, "=", 2)
		exports[idx] = fmt.Sprintf("export %s=%s", pair[0], escapeSingleQuote(pair[1]))
	}
	temp := writeTemporaryFile(append(exports, command), "\n")
	defer os.Remove(temp)
	tmuxArgs = append(tmuxArgs, "sh", temp)

	// Take the output
	go func() {
		outputFile, err := os.OpenFile(output, os.O_RDONLY, 0)
		if err != nil {
			return
		}
		if opts.Output == nil {
			io.Copy(os.Stdout, outputFile)
		} else {
			reader := bufio.NewReader(outputFile)
			sep := opts.PrintSep[0]
			for {
				item, err := reader.ReadString(sep)
				if err != nil {
					break
				}
				opts.Output <- item
			}
		}

		outputFile.Close()
	}()

	cmd := exec.Command("tmux", tmuxArgs...)
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			code := exitError.ExitCode()
			if code == ExitBecome {
				data, err := os.ReadFile(temp)
				if err != nil {
					return ExitError, err
				}
				elems := strings.Split(string(data), "\x00")
				if len(elems) < 1 {
					return ExitError, errors.New("invalid become command")
				}
				command := elems[0]
				env := []string{}
				if len(elems) > 1 {
					env = elems[1:]
				}
				os.Remove(temp)
				executor := util.NewExecutor(opts.WithShell)
				executor.Become(tui.TtyIn(), env, command)
			}
			return code, err
		}
	}

	return ExitOk, nil
}
