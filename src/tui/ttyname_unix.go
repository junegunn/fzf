//go:build !windows

package tui

import (
	"os"
	"sync/atomic"
	"syscall"
)

var devPrefixes = [...]string{"/dev/pts/", "/dev/"}

var tty atomic.Value

func ttyname() string {
	if cached := tty.Load(); cached != nil {
		return cached.(string)
	}

	var stderr syscall.Stat_t
	if syscall.Fstat(2, &stderr) != nil {
		return ""
	}

	for _, prefix := range devPrefixes {
		files, err := os.ReadDir(prefix)
		if err != nil {
			continue
		}

		for _, file := range files {
			info, err := file.Info()
			if err != nil {
				continue
			}
			if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Rdev == stderr.Rdev {
				value := prefix + file.Name()
				tty.Store(value)
				return value
			}
		}
	}
	return ""
}

// TtyIn returns terminal device to read user input
func TtyIn() (*os.File, error) {
	return openTtyIn()
}

// TtyIn returns terminal device to write to
func TtyOut() (*os.File, error) {
	return openTtyOut()
}
