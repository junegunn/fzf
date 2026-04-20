Building fzf
============

Build instructions
------------------

### Prerequisites

- Go 1.23 or above

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

> [!WARNING]
> Makefile uses git commands to determine the version and the revision
> information for `fzf --version`. So if you're building fzf from an
> environment where its git information is not available, you have to manually
> set `$FZF_VERSION` and `$FZF_REVISION`.
>
> e.g. `FZF_VERSION=0.24.0 FZF_REVISION=tarball make`

> [!TIP]
> To build fzf with profiling options enabled, set `TAGS=pprof`
>
> ```sh
> TAGS=pprof make clean install
> fzf --profile-cpu /tmp/cpu.pprof --profile-mem /tmp/mem.pprof \
>     --profile-block /tmp/block.pprof --profile-mutex /tmp/mutex.pprof
> ```

Running tests
-------------

```sh
# Run go unit tests
make test

# Run integration tests (requires to be on tmux)
make itest

# Run a single test case
ruby test/runner.rb --name test_something
```

Third-party libraries used
--------------------------

- [rivo/uniseg](https://github.com/rivo/uniseg)
    - Licensed under [MIT](https://raw.githubusercontent.com/rivo/uniseg/master/LICENSE.txt)
- [mattn/go-shellwords](https://github.com/mattn/go-shellwords)
    - Licensed under [MIT](http://mattn.mit-license.org)
- [mattn/go-isatty](https://github.com/mattn/go-isatty)
    - Licensed under [MIT](http://mattn.mit-license.org)
- [tcell](https://github.com/gdamore/tcell)
    - Licensed under [Apache License 2.0](https://github.com/gdamore/tcell/blob/master/LICENSE)
- [fastwalk](https://github.com/charlievieth/fastwalk)
    - Licensed under [MIT](https://raw.githubusercontent.com/charlievieth/fastwalk/master/LICENSE)

License
-------

[MIT](LICENSE)
