package main

import (
	"github.com/junegunn/fzf/src"
	"github.com/junegunn/fzf/src/protector"
)

var version string = "0.26"
var revision string = "devel"

func main() {
	protector.Protect()
	fzf.Run(fzf.ParseOptions(), version, revision)
}
