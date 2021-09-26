// +build windows

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
	escaped := strings.Replace(entry, `\`, `\\`, -1)
	escaped = `"` + strings.Replace(escaped, `"`, `\"`, -1) + `"`
	r, _ := regexp.Compile(`[&|<>()@^%!"]`)
	return r.ReplaceAllStringFunc(escaped, func(match string) string {
		return "^" + match
	})
}
