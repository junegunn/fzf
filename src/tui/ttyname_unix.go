//go:build !windows

package tui

import (
	"os"
	"syscall"
)

var devPrefixes = [...]string{"/dev/pts/", "/dev/"}

func ttyname() string {
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
				return prefix + file.Name()
			}
		}
	}
	return ""
}

// TtyIn returns terminal device to be used as STDIN, falls back to os.Stdin
func TtyIn() *os.File {
	if in, err := openTtyIn(); err == nil {
		return in
	}
	return os.Stdin
}
