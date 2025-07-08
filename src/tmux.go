package fzf

import (
	"os"
	"os/exec"

	"github.com/junegunn/fzf/src/tui"
)

func runTmux(args []string, opts *Options) (int, error) {
	// Prepare arguments
	fzf, rest := args[0], args[1:]
	args = []string{"--bind=ctrl-z:ignore"}
	if !opts.Tmux.border && (opts.BorderShape == tui.BorderUndefined || opts.BorderShape == tui.BorderLine) {
		// We append --border option at the end, because `--style=full:STYLE`
		// may have changed the default border style.
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
	argStr += ` --no-tmux --no-height`

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
	tmuxArgs := []string{"display-popup", "-E", "-d", dir}
	if !opts.Tmux.border {
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
