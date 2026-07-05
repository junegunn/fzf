package fzf

import (
	"os/exec"
)

func runZellij(args []string, opts *Options) (int, error) {
	argStr, dir := popupArgStr(args, opts)

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
