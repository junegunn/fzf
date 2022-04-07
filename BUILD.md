Building fzf
============

Build instructions
------------------

### Prerequisites

- Go 1.17 or above

### Using Makefile

```sh
# Build fzf binary for your platform in target
make

# Build fzf binary and copy it to bin directory
make install

# Build fzf binaries and archives for all platforms using goreleaser
make build

# Publish GitHub release
make release
```

> :warning: Makefile uses git commands to determine the version and the
> revision information for `fzf --version`. So if you're building fzf from an
> environment where its git information is not available, you have to manually
> set `$FZF_VERSION` and `$FZF_REVISION`.
>
> e.g. `FZF_VERSION=0.24.0 FZF_REVISION=tarball make`

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
