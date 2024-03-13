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

//go:embed shell/key-bindings.fish
var fishKeyBindings []byte

func main() {
	protector.Protect()
	options := fzf.ParseOptions()
	if options.Bash {
		fmt.Println("### key-bindings.bash ###")
		fmt.Println(string(bashKeyBindings))
		fmt.Println("### completion.bash ###")
		fmt.Println(string(bashCompletion))
		return
	}
	if options.Zsh {
		fmt.Println("### key-bindings.zsh ###")
		fmt.Println(string(zshKeyBindings))
		fmt.Println("### completion.zsh ###")
		fmt.Println(string(zshCompletion))
		return
	}
	if options.Fish {
		fmt.Println("### key-bindings.fish ###")
		fmt.Println(string(fishKeyBindings))
		fmt.Println("fzf_key_bindings")
		return
	}
	fzf.Run(options, version, revision)
}
