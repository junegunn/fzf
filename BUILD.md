Building fzf
============

Build instructions
------------------

### Prerequisites

- `go` executable in $PATH

### Using Makefile

Makefile will set up and use its own `$GOPATH` under the project root.

```sh
# Source files are located in src directory
cd src

# Build fzf binary for your platform in src/fzf
make

# Build fzf binary and copy it to bin directory
make install

# Build 32-bit and 64-bit executables and tarballs
make release

# Build executables and tarballs for Linux using Docker
make linux
```

### Using `go get`

Alternatively, you can build fzf directly with `go get` command without
cloning the repository.

```sh
go get -u github.com/junegunn/fzf/src/fzf
```

Build options
-------------

### With ncurses 6

The official binaries of fzf are built with ncurses 5 because it's widely
supported by different platforms. However ncurses 5 is old and has a number of
limitations.

1. Does not support more than 256 color pairs (See [357][357])
2. Does not support italics
3. Does not support 24-bit color

[357]: https://github.com/junegunn/fzf/issues/357

But you can manually build fzf with ncurses 6 to overcome some of these
limitations. ncurses 6 supports up to 32767 color pairs (1), and supports
italics (2). To build fzf with ncurses 6, you have to install it first. On
macOS, you can use Homebrew to install it.

```sh
brew install homebrew/dupes/ncurses
LDFLAGS="-L/usr/local/opt/ncurses/lib" make install
```

### With tcell

[tcell][tcell] is a portable alternative to ncurses and we currently use it to
build Windows binaries. tcell has many benefits but most importantly, it
supports 24-bit colors. To build fzf with tcell:

```sh
TAGS=tcell make install
```

However, note that tcell has its own issues.

- Poor rendering performance compared to ncurses
- Does not support bracketed-paste mode
- Does not support italics unlike ncurses 6
- Some wide characters are not correctly displayed

Third-party libraries used
--------------------------

- [ncurses][ncurses]
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

[install]: https://github.com/junegunn/fzf#installation
[go]:      https://golang.org/
[gil]:     http://en.wikipedia.org/wiki/Global_Interpreter_Lock
[ncurses]: https://www.gnu.org/software/ncurses/
[req]:     http://golang.org/doc/install
[tcell]:   https://github.com/gdamore/tcell
