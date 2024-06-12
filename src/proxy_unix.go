//go:build !windows

package fzf

import (
	"io"
	"os"

	"golang.org/x/sys/unix"
)

func sh() (string, error) {
	return "sh", nil
}

func mkfifo(path string, mode uint32) (string, error) {
	return path, unix.Mkfifo(path, mode)
}

func withOutputPipe(output string, task func(io.ReadCloser)) error {
	outputFile, err := os.OpenFile(output, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	task(outputFile)
	outputFile.Close()
	return nil
}

func withInputPipe(input string, task func(io.WriteCloser)) error {
	inputFile, err := os.OpenFile(input, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	task(inputFile)
	inputFile.Close()
	return nil
}
