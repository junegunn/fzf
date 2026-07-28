package fzf

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/junegunn/fzf/src/tui"
	"github.com/junegunn/fzf/src/util"
)

const becomeSuffix = ".become"

// Withheld from the environment replay below, so an outer value cannot
// override what the floating pane set
const internalEnvPrefix = "__FZF_INTERNAL_"

func escapeSingleQuote(str string) string {
	return "'" + strings.ReplaceAll(str, "'", "'\\''") + "'"
}

// Read and remove, so preview and execute commands do not inherit it
func takeEnv(name string) string {
	value := os.Getenv(name)
	os.Unsetenv(name)
	return value
}

// Applies label updates off the event loop, which would block on the fork
type labelUpdater struct {
	set     func(string)
	mu      sync.Mutex
	pending *string
	running bool
}

// A queue of one: a burst collapses to the newest label. A goroutine per
// update could apply them out of order
func (u *labelUpdater) update(label string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.pending = &label
	if u.running {
		return
	}
	u.running = true
	go u.run()
}

// Ends once caught up, so nothing needs cancelling when Run returns
func (u *labelUpdater) run() {
	for {
		u.mu.Lock()
		label := u.pending
		if label == nil {
			u.running = false
			u.mu.Unlock()
			return
		}
		u.pending = nil
		u.mu.Unlock()
		u.set(*label)
	}
}

// Updates the label on the native border, or nil when it is not fzf's to update
func nativeLabelSetter() func(string) {
	// Read both, so neither is left behind
	tmuxPane, zellijPane := takeEnv(tmuxBorderLabelEnv), takeEnv(zellijBorderLabelEnv)
	var set func(string)
	switch {
	case tmuxPane != "":
		set = func(label string) { setTmuxBorderLabel(tmuxPane, label) }
	case zellijPane != "":
		set = func(label string) { setZellijBorderLabel(zellijPane, label) }
	default:
		return nil
	}
	updater := &labelUpdater{set: set}
	return updater.update
}

func popupArgStr(args []string, opts *Options) (string, string) {
	fzf, rest := args[0], args[1:]
	args = []string{"--bind=ctrl-z:ignore"}
	if !opts.Tmux.border && (opts.BorderShape == tui.BorderUndefined || opts.BorderShape == tui.BorderLine) {
		if tui.DefaultBorderShape == tui.BorderRounded {
			rest = append(rest, "--border=rounded")
		} else {
			rest = append(rest, "--border=sharp")
		}
	}
	if opts.Tmux.border && opts.Margin == defaultMargin() {
		args = append(args, "--margin=0,1")
	}
	argStr := escapeSingleQuote(fzf)
	for _, arg := range append(args, rest...) {
		argStr += " " + escapeSingleQuote(arg)
	}
	argStr += ` --no-popup --no-height`

	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}
	return argStr, dir
}

func fifo(name string) (string, error) {
	ns := time.Now().UnixNano()
	output := filepath.Join(os.TempDir(), fmt.Sprintf("fzf-%s-%d", name, ns))
	output, err := mkfifo(output, 0600)
	if err != nil {
		return output, err
	}
	return output, nil
}

func runProxy(commandPrefix string, cmdBuilder func(temp string, needBash bool) (*exec.Cmd, error), opts *Options, withExports bool) (int, error) {
	output, err := fifo("proxy-output")
	if err != nil {
		return ExitError, err
	}
	defer os.Remove(output)

	// Take the output
	go func() {
		withOutputPipe(output, func(outputFile io.ReadCloser) {
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
		})
	}()

	var command, input string
	commandPrefix += ` --no-force-tty-in --proxy-script "$0"`
	if opts.Input == nil && (opts.ForceTtyIn || util.IsTty(os.Stdin)) {
		command = fmt.Sprintf(`%s > %q`, commandPrefix, output)
	} else {
		input, err = fifo("proxy-input")
		if err != nil {
			return ExitError, err
		}
		defer os.Remove(input)

		go func() {
			withInputPipe(input, func(inputFile io.WriteCloser) {
				if opts.Input == nil {
					io.Copy(inputFile, os.Stdin)
				} else {
					for item := range opts.Input {
						fmt.Fprint(inputFile, item+opts.PrintSep)
					}
				}
			})
		}()

		if withExports {
			command = fmt.Sprintf(`%s < %q > %q`, commandPrefix, input, output)
		} else {
			// For mintty: cannot directly read named pipe from Go code
			command = fmt.Sprintf(`command cat %q | %s > %q`, input, commandPrefix, output)
		}
	}

	// Write the command to a temporary file and run it with sh to ensure POSIX compliance.
	var exports []string
	needBash := false
	if withExports {
		// Nullify FZF_DEFAULT_* variables as tmux popup may inject them even when undefined.
		exports = []string{"FZF_DEFAULT_COMMAND=", "FZF_DEFAULT_OPTS=", "FZF_DEFAULT_OPTS_FILE="}
		validIdentifier := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
		for _, pairStr := range os.Environ() {
			pair := strings.SplitN(pairStr, "=", 2)
			if strings.HasPrefix(pair[0], internalEnvPrefix) {
				continue
			}
			if validIdentifier.MatchString(pair[0]) {
				exports = append(exports, fmt.Sprintf("export %s=%s", pair[0], escapeSingleQuote(pair[1])))
			} else if strings.HasPrefix(pair[0], "BASH_FUNC_") && strings.HasSuffix(pair[0], "%%") {
				name := pair[0][10 : len(pair[0])-2]
				exports = append(exports, name+pair[1])
				exports = append(exports, "export -f "+name)
				needBash = true
			}
		}
	}
	temp := WriteTemporaryFile(append(exports, command), "\n")
	defer os.Remove(temp)

	// Pre-create the become file so that another user on a shared TMPDIR
	// cannot plant a file or a symbolic link at the predictable path while
	// fzf is running
	becomeFile := temp + becomeSuffix
	becomeF, err := os.OpenFile(becomeFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return ExitError, err
	}
	becomeF.Close()
	defer os.Remove(becomeFile)

	cmd, err := cmdBuilder(temp, needBash)
	if err != nil {
		return ExitError, err
	}
	cmd.Stderr = os.Stderr
	intChan := make(chan os.Signal, 1)
	defer close(intChan)
	go func() {
		if sig, valid := <-intChan; valid {
			cmd.Process.Signal(sig)
		}
	}()
	signal.Notify(intChan, os.Interrupt)
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			code := exitError.ExitCode()
			if code == ExitBecome {
				data, err := os.ReadFile(becomeFile)
				os.Remove(becomeFile)
				if err != nil {
					return ExitError, err
				}
				if len(data) == 0 {
					return ExitError, errors.New("invalid become command")
				}
				elems := strings.Split(string(data), "\x00")
				command := elems[0]
				env := []string{}
				if len(elems) > 1 {
					env = elems[1:]
				}
				executor := util.NewExecutor(opts.WithShell)
				ttyin, err := tui.TtyIn(opts.TtyDefault)
				if err != nil {
					return ExitError, err
				}
				os.Remove(temp)
				os.Remove(input)
				os.Remove(output)
				executor.Become(ttyin, env, command)
			}
			return code, err
		}
	}

	return ExitOk, nil
}
