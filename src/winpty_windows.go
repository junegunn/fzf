//go:build windows

package fzf

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/junegunn/fzf/src/util"
)

func isMintty345() bool {
	return util.CompareVersions(os.Getenv("TERM_PROGRAM_VERSION"), "3.4.5") >= 0
}

func needWinpty(opts *Options) bool {
	if os.Getenv("TERM_PROGRAM") != "mintty" {
		return false
	}
	if isMintty345() {
		/*
		 See: https://github.com/junegunn/fzf/issues/3809

		 "MSYS=enable_pcon" allows fzf to run properly on mintty 3.4.5 or later.
		*/
		if strings.Contains(os.Getenv("MSYS"), "enable_pcon") {
			return false
		}

		// Setting the environment variable here unfortunately doesn't help,
		// so we need to start a child process with "MSYS=enable_pcon"
		//   os.Setenv("MSYS", "enable_pcon")
		return true
	}
	if opts.NoWinpty {
		return false
	}
	if _, err := exec.LookPath("winpty"); err != nil {
		return false
	}
	return true
}

func runWinpty(args []string, opts *Options) (int, error) {
	argStr := escapeSingleQuote(args[0])
	for _, arg := range args[1:] {
		argStr += " " + escapeSingleQuote(arg)
	}
	argStr += ` --no-winpty`

	if isMintty345() {
		return runProxy(argStr, func(temp string, needBash bool) (*exec.Cmd, error) {
			sh, err := sh(needBash)
			if err != nil {
				return nil, err
			}

			cmd := exec.Command(sh, temp)
			cmd.Env = append(os.Environ(), "MSYS=enable_pcon")
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd, nil
		}, opts, false)
	}

	return runProxy(argStr, func(temp string, needBash bool) (*exec.Cmd, error) {
		sh, err := sh(needBash)
		if err != nil {
			return nil, err
		}

		cmd := exec.Command(sh, "-c", fmt.Sprintf(`winpty < /dev/tty > /dev/tty -- sh %q`, temp))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd, nil
	}, opts, false)
}
