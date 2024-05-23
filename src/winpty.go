//go:build !windows

package fzf

import "errors"

func needWinpty(_ *Options) bool {
	return false
}

func runWinpty(_ []string, _ *Options) (int, error) {
	return ExitError, errors.New("Not supported")
}
