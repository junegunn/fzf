package fzf

import (
	"os"
	"os/exec"

	"github.com/junegunn/fzf/src/tui"
)

func runZellij(args []string, opts *Options) (int, error) {
	// Prepare arguments
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

	// Get current directory
	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}

	zellijArgs := []string{
		"run", "--floating", "--close-on-exit", "--block-until-exit",
		"--cwd", dir,
	}
	if !opts.Tmux.border {
		zellijArgs = append(zellijArgs, "--borderless", "true")
	}
	switch opts.Tmux.position {
	case posUp:
		zellijArgs = append(zellijArgs, "-y", "0")
	case posDown:
		zellijArgs = append(zellijArgs, "-y", "9999")
	case posLeft:
		zellijArgs = append(zellijArgs, "-x", "0")
	case posRight:
		zellijArgs = append(zellijArgs, "-x", "9999")
	case posCenter:
		// Zellij centers floating panes by default
	}
	zellijArgs = append(zellijArgs, "--width", opts.Tmux.width.String())
	zellijArgs = append(zellijArgs, "--height", opts.Tmux.height.String())
	zellijArgs = append(zellijArgs, "--")

	return runProxy(argStr, func(temp string, needBash bool) (*exec.Cmd, error) {
		sh, err := sh(needBash)
		if err != nil {
			return nil, err
		}
		zellijArgs = append(zellijArgs, sh, temp)
		return exec.Command("zellij", zellijArgs...), nil
	}, opts, true)
}
