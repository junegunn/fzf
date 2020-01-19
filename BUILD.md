Building fzf
============

Build instructions
------------------

### Prerequisites

- Go 1.11 or above

### Using Makefile

```sh
# Build fzf binary for your platform in target
make

# Build fzf binary and copy it to bin directory
make install

# Build 32-bit and 64-bit executables and tarballs in target
make release

# Make release archives for all supported platforms in target
make release-all
```

### Using `go get`

Alternatively, you can build fzf directly with `go get` command without
manually cloning the repository.

```sh
go get -u github.com/junegunn/fzf
```

Third-party libraries used
--------------------------

- [mattn/go-runewidth](https://github.com/mattn/go-runewidth)
    - Licensed under [MIT](http://mattn.mit-license.org)
- [mattn/go-shellwords](https://github.com/mattn/go-shellwords)
    - Licensed under [MIT](http://mattn.mit-license.org)
- [mattn/go-isatty](https://github.com/mattn/go-isatty)
    - Licensed under [MIT](http://mattn.mit-license.org)
- [tcell](https://github.com/gdamore/tcell)
    - Licensed under [Apache License 2.0](https://github.com/gdamore/tcell/blob/master/LICENSE)

License
-------

[MIT](LICENSE)
