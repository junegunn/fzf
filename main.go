package main

import (
	_ "embed"
	"fmt"

	fzf "github.com/junegunn/fzf/src"
	"github.com/junegunn/fzf/src/protector"
)

var version string = "0.47"
var revision string = "devel"

//go:embed shell/key-bindings.bash
var bashKeyBindings []byte

//go:embed shell/completion.bash
var bashCompletion []byte

//go:embed shell/key-bindings.zsh
var zshKeyBindings []byte

//go:embed shell/completion.zsh
var zshCompletion []byte

func main() {
	protector.Protect()
	options := fzf.ParseOptions()
	if options.Bash {
		fmt.Println(string(bashKeyBindings))
		fmt.Println(string(bashCompletion))
		return
	}
	if options.Zsh {
		fmt.Println(string(zshKeyBindings))
		fmt.Println(string(zshCompletion))
		return
	}
	fzf.Run(options, version, revision)
}
