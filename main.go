package main

import (
	"github.com/junegunn/fzf/src"
	"github.com/junegunn/fzf/src/protector"
)

var version string
var revision string

func main() {
	protector.Protect()
	fzf.Run(fzf.ParseOptions(), version, revision)
}
