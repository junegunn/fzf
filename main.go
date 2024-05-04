package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	fzf "github.com/junegunn/fzf/src"
	"github.com/junegunn/fzf/src/protector"
	"github.com/junegunn/fzf/src/util"
)

var version string = "0.51"
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

func errorExit(msg string) {
	os.Stderr.WriteString(msg + "\n")
	os.Exit(fzf.ExitError)
}

func main() {
	protector.Protect()

	options, err := fzf.ParseOptions(true, os.Args[1:])
	if err != nil {
		errorExit(err.Error())
		return
	}
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
	code, err := fzf.Run(options, version, revision)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
	}
	util.Exit(code)
}
