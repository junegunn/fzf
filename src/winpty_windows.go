//go:build windows

package fzf

import (
	"fmt"
	"os"
	"os/exec"
)

func runWinpty(args []string, opts *Options) (int, error) {
	sh, err := sh()
	if err != nil {
		return ExitError, err
	}

	argStr := escapeSingleQuote(args[0])
	for _, arg := range args[1:] {
		argStr += " " + escapeSingleQuote(arg)
	}
	argStr += ` --no-winpty --no-height`

	return runProxy(argStr, func(temp string) *exec.Cmd {
		cmd := exec.Command(sh, "-c", fmt.Sprintf(`winpty < /dev/tty > /dev/tty -- sh %q`, temp))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd
	}, opts, false)
}
