//go:build windows

package tui

import "os"

func ttyname() string {
	return ""
}

// TtyIn on Windows returns os.Stdin
func TtyIn() *os.File {
	return os.Stdin
}
