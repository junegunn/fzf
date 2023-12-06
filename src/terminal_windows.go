//go:build windows

package fzf

import (
	"os"
	"regexp"
	"strings"
)

func notifyOnResize(resizeChan chan<- os.Signal) {
	// TODO
}

func notifyStop(p *os.Process) {
	// NOOP
}

func notifyOnCont(resizeChan chan<- os.Signal) {
	// NOOP
}

func quoteEntry(entry string) string {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "cmd"
	}

	if strings.Contains(shell, "cmd") {
		// backslash escaping is done here for applications
		// (see ripgrep test case in terminal_test.go#TestWindowsCommands)
		escaped := strings.Replace(entry, `\`, `\\`, -1)
		escaped = `"` + strings.Replace(escaped, `"`, `\"`, -1) + `"`
		// caret is the escape character for cmd shell
		r, _ := regexp.Compile(`[&|<>()@^%!"]`)
		return r.ReplaceAllStringFunc(escaped, func(match string) string {
			return "^" + match
		})
	} else if strings.Contains(shell, "pwsh") || strings.Contains(shell, "powershell") {
		escaped := strings.Replace(entry, `"`, `\"`, -1)
		return "'" + strings.Replace(escaped, "'", "''", -1) + "'"
	} else {
		return "'" + strings.Replace(entry, "'", "'\\''", -1) + "'"
	}
}
