package fzf

import (
	"os"
	"os/exec"

	"github.com/junegunn/fzf/src/tui"
)

func runTmux(args []string, opts *Options) (int, error) {
	// Prepare arguments
	fzf := args[0]
	args = append([]string{"--bind=ctrl-z:ignore"}, args[1:]...)
	if opts.BorderShape == tui.BorderUndefined {
		args = append(args, "--border")
	}
	argStr := escapeSingleQuote(fzf)
	for _, arg := range args {
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
	tmuxArgs := []string{"display-popup", "-E", "-B", "-d", dir}
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

	return runProxy(argStr, func(temp string) *exec.Cmd {
		sh, _ := sh()
		tmuxArgs = append(tmuxArgs, sh, temp)
		return exec.Command("tmux", tmuxArgs...)
	}, opts, true)
}
