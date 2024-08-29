package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"strings"

	fzf "github.com/junegunn/fzf/src"
	"github.com/junegunn/fzf/src/protector"
)

var version = "0.55"
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

//go:embed man/man1/fzf.1
var manPage []byte

func printScript(label string, content []byte) {
	fmt.Println("### " + label + " ###")
	fmt.Println(strings.TrimSpace(string(content)))
	fmt.Println("### end: " + label + " ###")
}

func exit(code int, err error) {
	if code == fzf.ExitError && err != nil {
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
	if options.Man {
		file := fzf.WriteTemporaryFile([]string{string(manPage)}, "\n")
		if len(file) == 0 {
			fmt.Print(string(manPage))
			return
		}
		defer os.Remove(file)
		cmd := exec.Command("man", file)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			fmt.Print(string(manPage))
		}
		return
	}

	code, err := fzf.Run(options)
	exit(code, err)
}
