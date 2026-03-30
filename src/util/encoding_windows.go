//go:build windows

package util

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/codepage"
)

var (
	// Cached console output codepage
	consoleCodepage     uint32
	consoleCodepageInit bool
	consoleEncoder      encoding.Encoding
)

// GetConsoleOutputCodepage returns the Windows console output codepage.
// Returns 0 if the console is not available.
func GetConsoleOutputCodepage() uint32 {
	if consoleCodepageInit {
		return consoleCodepage
	}
	consoleCodepageInit = true

	// Try to get the console output codepage
	// We use GetConsoleOutputCP() which returns the output codepage of the
	// console buffer associated with the calling process.
	cp := windows.GetConsoleOutputCP()
	consoleCodepage = uint32(cp)
	return consoleCodepage
}

// GetConsoleEncoder returns an encoder for the console output codepage.
// Returns nil if the codepage cannot be determined or is UTF-8.
func GetConsoleEncoder() encoding.Encoding {
	if consoleEncoder != nil {
		return consoleEncoder
	}

	cp := GetConsoleOutputCodepage()
	if cp == 0 || cp == 65001 { // 65001 is UTF-8
		return nil
	}

	// Get the codepage encoder
	consoleEncoder = codepage.CodePage(cp)
	return consoleEncoder
}

// WriteWithConsoleEncoding writes the string to the writer with console encoding
// if stdout is redirected on Windows. This is needed because Go defaults to UTF-8
// for non-console output, but PowerShell expects the console's codepage.
//
// When output is redirected (e.g., $x = fzf in PowerShell), the bytes written
// to stdout are UTF-8 encoded, but PowerShell decodes them using the console's
// output codepage. This causes encoding issues for non-ASCII characters.
//
// This function detects if stdout is redirected and, if so, converts the output
// to the console's output codepage before writing.
func WriteWithConsoleEncoding(w io.Writer, str string) (int, error) {
	// If writing to stdout and stdout is redirected, use console encoding
	if f, ok := w.(*os.File); ok && f == os.Stdout && !IsTty(os.Stdout) {
		encoder := GetConsoleEncoder()
		if encoder != nil {
			// Encode the string to the console codepage
			encoded, err := encoder.NewEncoder().String(str)
			if err != nil {
				// If encoding fails, fall back to original string
				return fmt.Fprint(w, str)
			}
			return fmt.Fprint(w, encoded)
		}
	}
	return fmt.Fprint(w, str)
}

// PrintWithConsoleEncoding prints the string to stdout with console encoding
// if stdout is redirected on Windows.
func PrintWithConsoleEncoding(str string) (int, error) {
	return WriteWithConsoleEncoding(os.Stdout, str)
}

// PrintlnWithConsoleEncoding prints the string to stdout with a newline,
// using console encoding if stdout is redirected on Windows.
func PrintlnWithConsoleEncoding(str string) (int, error) {
	return PrintWithConsoleEncodingSep(str, "\n")
}

// PrintWithConsoleEncodingSep prints the string to stdout with the given separator,
// using console encoding if stdout is redirected on Windows.
func PrintWithConsoleEncodingSep(str string, sep string) (int, error) {
	// If writing to stdout and stdout is redirected, use console encoding
	if !IsTty(os.Stdout) {
		encoder := GetConsoleEncoder()
		if encoder != nil {
			// Encode the string to the console codepage
			encoded, err := encoder.NewEncoder().String(str)
			if err != nil {
				// If encoding fails, fall back to original string
				return fmt.Print(str, sep)
			}
			return fmt.Print(encoded, sep)
		}
	}
	return fmt.Print(str, sep)
}