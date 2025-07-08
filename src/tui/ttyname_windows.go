//go:build windows

package tui

import (
	"os"
)

func ttyname() string {
	return ""
}

// TtyIn on Windows returns os.Stdin
func TtyIn(ttyDefault string) (*os.File, error) {
	return os.Stdin, nil
}

// TtyOut on Windows returns nil
func TtyOut(ttyDefault string) (*os.File, error) {
	return nil, nil
}
