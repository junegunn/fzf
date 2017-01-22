fzf in Go
=========

<img src="https://cloud.githubusercontent.com/assets/700826/5725028/028ea834-9b93-11e4-9198-43088c3f295d.gif" height="463" alt="fzf in go">

This directory contains the source code for the new fzf implementation in
[Go][go].

Upgrade from Ruby version
-------------------------

The install script has been updated to download the right binary for your
system. If you already have installed fzf, simply git-pull the repository and
rerun the install script.

```sh
cd ~/.fzf
git pull
./install
```

Otherwise, follow [the instruction][install] as before. You can also install
fzf using Homebrew if you prefer that way.

Motivations
-----------

### No Ruby dependency

There have always been complaints about fzf being a Ruby script. To make
matters worse, Ruby 2.1 removed ncurses binding from its standard libary.
Because of the change, users running Ruby 2.1 or above are forced to build C
extensions of curses gem to meet the requirement of fzf. The new Go version
will be distributed as an executable binary so it will be much more accessible
and should be easier to setup.

### Performance

Many people have been surprised to see how fast fzf is even when it was
written in Ruby. It stays quite responsive even for 100k+ lines, which is
well above the size of the usual input.

The new Go version, of course, is significantly faster than that. It has all
the performance optimization techniques used in Ruby implementation and more.
It also doesn't suffer from [GIL][gil], so the search performance scales
proportional to the number of CPU cores. On my MacBook Pro (Mid 2012), the new
version was shown to be an order of magnitude faster on certain cases. It also
starts much faster though the difference may not be noticeable.

Build
-----

See [BUILD.md](../BUILD.md)

Test
----

Unit tests can be run with `make test`. Integration tests are written in Ruby
script that should be run on tmux.

```sh
cd src

# Unit tests
make test

# Integration tests
ruby ../test/test_go.rb

# Build binary for the platform
make

# Install the executable to ../bin directory
make install

# Make release archives
make release

# Make release archives for all supported platforms
make release-all
```

Third-party libraries used
--------------------------

- ~[ncurses][ncurses]~
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
