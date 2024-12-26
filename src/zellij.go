package fzf

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/junegunn/fzf/src/tui"
)

func runZellij(args []string, opts *Options) (int, error) {
	// Prepare arguments
	fzf := args[0]
	args = append([]string{"--bind=ctrl-z:ignore"}, args[1:]...)
	if opts.BorderShape == tui.BorderUndefined {
		args = append(args, "--no-border")
	}
	argStr := escapeSingleQuote(fzf)
	for _, arg := range args {
		argStr += " " + escapeSingleQuote(arg)
	}
	argStr += ` --no-popup --no-height`

	// Get current directory
	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}

	sh, err := sh(false)
	if err != nil {
		return ExitError, err
	}

	fifo, err := fifo("zellij-fifo")
	if err != nil {
		return ExitError, err
	}

	zopts := " --width " + opts.Tmux.width.String() + " --height " + opts.Tmux.height.String()
	centerX := func() {
		// TODO: Handle non-percent values
		if opts.Tmux.width.percent {
			x := (100.0 - opts.Tmux.width.size) / 2
			if x <= 0 {
				zopts += " -x0"
			} else {
				zopts += fmt.Sprintf(" -x%d%%", int(x))
			}
		} else if cols := os.Getenv("COLUMNS"); len(cols) > 0 {
			if w, e := strconv.Atoi(cols); e == nil {
				x := (float64(w) - opts.Tmux.width.size) / 2
				zopts += fmt.Sprintf(" -x%d", int(x))
			}
		}

	}
	centerY := func() {
		if opts.Tmux.height.percent {
			y := (100.0 - opts.Tmux.height.size) / 2
			if y <= 0 {
				zopts += " -y0"
			} else {
				zopts += fmt.Sprintf(" -y%d%%", int(y))
			}
		} else if lines := os.Getenv("LINES"); len(lines) > 0 {
			if h, e := strconv.Atoi(lines); e == nil {
				y := (float64(h) - opts.Tmux.height.size) / 2
				zopts += fmt.Sprintf(" -y%d", int(y))
			}
		}
	}
	switch opts.Tmux.position {
	case posUp:
		zopts += " -y0"
		centerX()
	case posDown:
		zopts += " -y9999"
		centerX()
	case posLeft:
		zopts += " -x0"
		centerY()
	case posRight:
		zopts += " -x9999"
		centerY()
	case posCenter:
		centerX()
		centerY()
	}

	lines := []string{
		"#!/bin/sh",
		fmt.Sprintf(`zellij run --name '' --floating --close-on-exit --cwd %s %s -- %s -c "%s $1; echo \$? > %s" || exit $?`, dir, zopts, sh, sh, fifo),
		fmt.Sprintf(`exit $(cat %s)`, fifo),
	}
	temptemp := WriteTemporaryFile(lines, "\n")
	defer os.Remove(temptemp)

	return runProxy(argStr, func(temp string, needBash bool) (*exec.Cmd, error) {
		return exec.Command(sh, temptemp, temp), nil
	}, opts, true)
}
