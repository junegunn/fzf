//go:build windows

package tui

import (
	"os"
)

func ttyname() string {
	return ""
}

// TtyIn on Windows returns os.Stdin
func TtyIn() (*os.File, error) {
	return os.Stdin, nil
}

// TtyIn on Windows returns nil
func TtyOut() (*os.File, error) {
	return nil, nil
}
