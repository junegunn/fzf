//go:build !windows

package fzf

import "errors"

func runWinpty(_ []string, _ *Options) (int, error) {
	return ExitError, errors.New("Not supported")
}
