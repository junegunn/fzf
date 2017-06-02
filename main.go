package main

import "github.com/junegunn/fzf/src"

var revision string

func main() {
	fzf.Run(fzf.ParseOptions(), revision)
}
