package main

import (
	_ "embed"
	"fmt"
	"strings"

	fzf "github.com/junegunn/fzf/src"
	"github.com/junegunn/fzf/src/protector"
)

var version string = "0.48"
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

func printScript(label string, content []byte) {
	fmt.Println("### " + label + " ###")
	fmt.Println(strings.TrimSpace(string(content)))
	fmt.Println("### end: " + label + " ###")
}

func main() {
	protector.Protect()
	options := fzf.ParseOptions()
	if options.Bash {
		printScript("key-bindings.bash", bashKeyBindings)
		printScript("completion.bash", bashCompletion)
		return
	}
	if options.Zsh {
		printScript("key-bindings.zsh", zshKeyBindings)
		printScript("completion.zsh", zshCompletion)
		return
	}
	if options.Fish {
		printScript("key-bindings.fish", fishKeyBindings)
		fmt.Println("fzf_key_bindings")
		return
	}
	fzf.Run(options, version, revision)
}
