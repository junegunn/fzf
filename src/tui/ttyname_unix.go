// +build !windows

package tui

import (
	"io/ioutil"
	"syscall"
)

var devPrefixes = [...]string{"/dev/pts/", "/dev/"}

func ttyname() string {
	var stderr syscall.Stat_t
	if syscall.Fstat(2, &stderr) != nil {
		return ""
	}

	for _, prefix := range devPrefixes {
		files, err := ioutil.ReadDir(prefix)
		if err != nil {
			continue
		}

		for _, file := range files {
			if stat, ok := file.Sys().(*syscall.Stat_t); ok && stat.Rdev == stderr.Rdev {
				return prefix + file.Name()
			}
		}
	}
	return ""
}
