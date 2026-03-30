//go:build !windows

package util

import (
	"fmt"
	"io"
	"os"
)

// GetConsoleOutputCodepage returns 0 on non-Windows platforms.
func GetConsoleOutputCodepage() uint32 {
	return 0
}

// GetConsoleEncoder returns nil on non-Windows platforms.
func GetConsoleEncoder() interface{} {
	return nil
}

// WriteWithConsoleEncoding writes the string to the writer directly on non-Windows platforms.
func WriteWithConsoleEncoding(w io.Writer, str string) (int, error) {
	return fmt.Fprint(w, str)
}

// PrintWithConsoleEncoding prints the string to stdout directly on non-Windows platforms.
func PrintWithConsoleEncoding(str string) (int, error) {
	return fmt.Fprint(os.Stdout, str)
}

// PrintlnWithConsoleEncoding prints the string to stdout with a newline on non-Windows platforms.
func PrintlnWithConsoleEncoding(str string) (int, error) {
	return fmt.Fprintln(os.Stdout, str)
}

// PrintWithConsoleEncodingSep prints the string to stdout with the given separator on non-Windows platforms.
func PrintWithConsoleEncodingSep(str string, sep string) (int, error) {
	return fmt.Print(str, sep)
}