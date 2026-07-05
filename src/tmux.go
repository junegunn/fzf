package fzf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Returns the size of the current window if the tmux server supports
// floating panes (tmux 3.7 or above)
func tmuxFloatingPaneInfo() (int, int, bool) {
	target := os.Getenv("TMUX_PANE")
	if target == "" {
		return 0, 0, false
	}
	// A single invocation for both checks. Cannot rely on the exit status;
	// tmux versions before 3.7 exit normally with empty output for an
	// unknown command name, so check the output instead.
	out, err := exec.Command("tmux", "display-message", "-p", "-t", target,
		"#{window_width} #{window_height}", ";", "list-commands", "new-pane").Output()
	if err != nil || !strings.Contains(string(out), "new-pane") {
		return 0, 0, false
	}
	var width, height int
	if _, err := fmt.Sscanf(string(out), "%d %d", &width, &height); err != nil {
		return 0, 0, false
	}
	// The window is too small to fit a floating pane of the minimum size
	if width < 3 || height < 3 {
		return 0, 0, false
	}
	return width, height, true
}

// Convert sizeSpec to the number of cells, clamped between the minimum
// footprint of 3, including the border, and the window size
func tmuxDim(spec sizeSpec, window int) int {
	dim := int(spec.size)
	if spec.percent {
		dim = window * dim / 100
	}
	return max(3, min(dim, window))
}

func runTmuxFloatingPane(argStr string, dir string, windowWidth int, windowHeight int, opts *Options) (int, error) {
	// Unlike display-popup, the size of a floating pane does not account for
	// the border around it, and the position is that of the content area. To
	// stay consistent with popups, treat the requested size as the total
	// footprint including the border.
	width := tmuxDim(opts.Tmux.width, windowWidth)
	height := tmuxDim(opts.Tmux.height, windowHeight)
	x := (windowWidth-width)/2 + 1
	y := (windowHeight-height)/2 + 1
	switch opts.Tmux.position {
	case posUp:
		y = 1
	case posDown:
		y = windowHeight - height + 1
	case posLeft:
		x = 1
	case posRight:
		x = windowWidth - width + 1
	}

	return runProxy(argStr, func(temp string, needBash bool) (*exec.Cmd, error) {
		sh, err := sh(needBash)
		if err != nil {
			return nil, err
		}
		// Unlike display-popup, new-pane does not block until the command
		// finishes, and it does not propagate the exit status. So we block on
		// a wait-for channel that the pane signals on completion, and pass
		// the exit status through a temporary file. A watchdog process
		// signals the same channel if the pane is closed abnormally
		// (e.g. kill-pane), in which case the file is not written.
		//
		// has-session is the liveness check because it fails when the target
		// pane is gone, while display-message succeeds even for a dead pane.
		signal := escapeSingleQuote("fzf-" + filepath.Base(temp))

		// Pre-create the exit status file so that another user on a shared
		// TMPDIR cannot plant a file or a symbolic link at the predictable
		// path while the pane is running
		codeFile := temp + ".code"
		f, err := os.OpenFile(codeFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, err
		}
		f.Close()
		code := escapeSingleQuote(codeFile)
		paneCmd := fmt.Sprintf("%s %s; echo $? > %s; tmux wait-for -S %s",
			escapeSingleQuote(sh), escapeSingleQuote(temp), code, signal)
		// Unzoom the window first; creating a floating pane over a zoomed
		// window crashes the tmux server on 3.7b, and newer versions of
		// tmux unzoom the window anyway. The pane should always close on
		// exit like a popup, even when remain-on-exit is on. set-option
		// targets the new pane as new-pane makes it the current pane.
		target := os.Getenv("TMUX_PANE")
		newPane := fmt.Sprintf(
			"tmux if -F -t %s '#{window_zoomed_flag}' %s ';' new-pane -P -F '#{pane_id}' -t %s -c %s -x %d -y %d -X %d -Y %d %s -c %s ';' set-option -p remain-on-exit off",
			escapeSingleQuote(target), escapeSingleQuote("resize-pane -Z -t "+target),
			escapeSingleQuote(target), escapeSingleQuote(dir), width-2, height-2, x, y,
			escapeSingleQuote(sh), escapeSingleQuote(paneCmd))
		// The pane is killed when the proxy process is interrupted or hung up,
		// like a popup dying with its client. wait-for runs in the background
		// and is awaited with the interruptible wait builtin so that the trap
		// can fire while blocked. The trap is installed before creating the
		// pane; a signal received during creation is deferred until the
		// command substitution completes, and the pane is killed right after.
		// An interrupted wait does not reap the waiter, so it is killed
		// along with the watchdog.
		script := fmt.Sprintf(`trap '[ -n "$id" ] && tmux kill-pane -t "$id" 2> /dev/null' INT TERM HUP
id=$(%s) || { status=$?; rm -f %s; exit "$status"; }
{ while tmux has-session -t "$id" 2> /dev/null; do sleep 1; done; tmux wait-for -S %s; } &
watchdog=$!
tmux wait-for %s &
waiter=$!
wait "$waiter"
kill "$watchdog" "$waiter" 2> /dev/null
wait 2> /dev/null
if [ -s %s ]; then code=$(cat %s); else code=130; fi
rm -f %s
exit "$code"`, newPane, code, signal, signal, code, code, code)
		return exec.Command(sh, "-c", script), nil
	}, opts, true)
}

func runTmux(args []string, opts *Options) (int, error) {
	// On tmux 3.7 or above, fzf runs in a floating pane instead of a popup.
	// A floating pane always has a native border, so 'border-native' is
	// implied. Give 'border-fzf' to fall back to a popup where fzf draws
	// its own border.
	if opts.Tmux.border != tmuxBorderFzf {
		if windowWidth, windowHeight, ok := tmuxFloatingPaneInfo(); ok {
			opts.Tmux.border = tmuxBorderNative
			argStr, dir := popupArgStr(args, opts)
			return runTmuxFloatingPane(argStr, dir, windowWidth, windowHeight, opts)
		}
	}

	argStr, dir := popupArgStr(args, opts)

	// Set tmux options for popup placement
	// C        Both    The centre of the terminal
	// R        -x      The right side of the terminal
	// P        Both    The bottom left of the pane
	// M        Both    The mouse position
	// W        Both    The window position on the status line
	// S        -y      The line above or below the status line
	tmuxArgs := []string{"display-popup", "-E", "-d", dir}
	if opts.Tmux.border != tmuxBorderNative {
		tmuxArgs = append(tmuxArgs, "-B")
	}
	switch opts.Tmux.position {
	case posUp:
		tmuxArgs = append(tmuxArgs, "-xC", "-y0")
	case posDown:
		tmuxArgs = append(tmuxArgs, "-xC", "-y9999")
	case posLeft:
		tmuxArgs = append(tmuxArgs, "-x0", "-yC")
	case posRight:
		tmuxArgs = append(tmuxArgs, "-xR", "-yC")
	case posCenter:
		tmuxArgs = append(tmuxArgs, "-xC", "-yC")
	}
	tmuxArgs = append(tmuxArgs, "-w"+opts.Tmux.width.String())
	tmuxArgs = append(tmuxArgs, "-h"+opts.Tmux.height.String())

	return runProxy(argStr, func(temp string, needBash bool) (*exec.Cmd, error) {
		sh, err := sh(needBash)
		if err != nil {
			return nil, err
		}
		tmuxArgs = append(tmuxArgs, sh, temp)
		return exec.Command("tmux", tmuxArgs...), nil
	}, opts, true)
}
