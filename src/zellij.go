package fzf

import (
	"os/exec"
)

func runZellij(args []string, opts *Options) (int, error) {
	// Use the native Zellij border by default, consistent with tmux, so that
	// the pane can be moved and resized with the mouse. Set before
	// popupArgStr so that it does not inject an fzf border. fzf draws its own
	// border instead when a border style is explicitly specified.
	if nativeBorder(opts) {
		opts.Tmux.border = true
	}
	argStr, dir := popupArgStr(args, opts)

	zellijArgs := []string{
		"run", "--floating", "--close-on-exit", "--block-until-exit",
		"--cwd", dir,
	}
	if !opts.Tmux.border {
		zellijArgs = append(zellijArgs, "--borderless", "true")
	} else {
		// Set --border-label as the name of the pane, displayed on the
		// native border. Empty when no label is given, to override the
		// default name (the running command). Passed as a distinct
		// argument, so no escaping is needed beyond stripping ANSI
		// sequences fzf would otherwise render itself. --border-label-pos
		// is ignored.
		label, _, _ := extractColor(opts.BorderLabel.label, nil, nil)
		zellijArgs = append(zellijArgs, "--name", label)
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
