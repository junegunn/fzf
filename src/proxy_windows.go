//go:build windows

package fzf

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
)

var shPath atomic.Value

func sh(bash bool) (string, error) {
	if cached := shPath.Load(); cached != nil {
		return cached.(string), nil
	}

	name := "sh"
	if bash {
		name = "bash"
	}
	cmd := exec.Command("cygpath", "-w", "/usr/bin/"+name)
	bytes, err := cmd.Output()
	if err != nil {
		return "", err
	}

	sh := strings.TrimSpace(string(bytes))
	shPath.Store(sh)
	return sh, nil
}

func mkfifo(path string, mode uint32) (string, error) {
	m := strconv.FormatUint(uint64(mode), 8)
	sh, err := sh(false)
	if err != nil {
		return path, err
	}
	cmd := exec.Command(sh, "-c", fmt.Sprintf(`command mkfifo -m %s %q`, m, path))
	if err := cmd.Run(); err != nil {
		return path, err
	}
	return path + ".lnk", nil
}

func withOutputPipe(output string, task func(io.ReadCloser)) error {
	sh, err := sh(false)
	if err != nil {
		return err
	}
	cmd := exec.Command(sh, "-c", fmt.Sprintf(`command cat %q`, output))
	outputFile, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	task(outputFile)
	cmd.Wait()
	return nil
}

func withInputPipe(input string, task func(io.WriteCloser)) error {
	sh, err := sh(false)
	if err != nil {
		return err
	}
	cmd := exec.Command(sh, "-c", fmt.Sprintf(`command cat - > %q`, input))
	inputFile, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	task(inputFile)
	inputFile.Close()
	cmd.Wait()
	return nil
}
