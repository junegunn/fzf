package main

import (
	"github.com/junegunn/fzf/src"
	"github.com/junegunn/fzf/src/protector"
)

var version string
var revision string

func main() {
	if len(version) == 0 {
		panic("Invalid build: version information missing")
	}
	protector.Protect()
	fzf.Run(fzf.ParseOptions(), version, revision)
}
