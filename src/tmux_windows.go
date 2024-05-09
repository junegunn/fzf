//go:build windows

package fzf

import (
	"os/exec"
	"strconv"
)

func mkfifo(path string, mode uint32) error {
	m := strconv.FormatUint(uint64(mode), 8)
	cmd := exec.Command("mkfifo", "-m", m, path)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
