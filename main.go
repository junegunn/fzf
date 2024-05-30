package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	fzf "github.com/junegunn/fzf/src"
	"github.com/junegunn/fzf/src/protector"
)

var version = "0.52"
var revision = "devel"

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

func exit(code int, err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	os.Exit(code)
}

func main() {
	protector.Protect()

	options, err := fzf.ParseOptions(true, os.Args[1:])
	if err != nil {
		exit(fzf.ExitError, err)
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
	if options.Help {
		fmt.Print(fzf.Usage)
		return
	}
	if options.Version {
		if len(revision) > 0 {
			fmt.Printf("%s (%s)\n", version, revision)
		} else {
			fmt.Println(version)
		}
		return
	}

	code, err := fzf.Run(options)
	exit(code, err)
}
