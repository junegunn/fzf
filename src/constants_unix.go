// +build !windows

package fzf

const (
	// Reader
	defaultCommand = `find . -path '*/\.*' -prune -o -type f -print -o -type l -print 2> /dev/null | sed s/^..//`
)
