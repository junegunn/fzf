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

Differences with Ruby version
-----------------------------

The Go version is designed to be perfectly compatible with the previous Ruby
version. The only behavioral difference is that the new version ignores the
numeric argument to `--sort=N` option and always sorts the result regardless
of the number of matches. The value was introduced to limit the response time
of the query, but the Go version is blazingly fast (almost instant response
even for 1M+ items) so I decided that it's no longer required.

System requirements
-------------------

Currently, prebuilt binaries are provided only for OS X and Linux. The install
script will fall back to the legacy Ruby version on the other systems, but if
you have Go 1.4 installed, you can try building it yourself.

However, as pointed out in [golang.org/doc/install][req], the Go version may
not run on CentOS/RHEL 5.x, and if that's the case, the install script will
choose the Ruby version instead.

The Go version depends on [ncurses][ncurses] and some Unix system calls, so it
shouldn't run natively on Windows at the moment. But it won't be impossible to
support Windows by falling back to a cross-platform alternative such as
[termbox][termbox] only on Windows. If you're interested in making fzf work on
Windows, please let me know.

Build
-----

```sh
# Build fzf executables and tarballs
make release

# Install the executable to ../bin directory
make install

# Build executables and tarballs for Linux using Docker
make linux
```

Contribution
------------

For the time being, I will not add or accept any new features until we can be
sure that the implementation is stable and we have a sufficient number of test
cases. However, fixes for obvious bugs and new test cases are welcome.

I also care much about the performance of the implementation, so please make
sure that your change does not result in performance regression. And please be
noted that we don't have a quantitative measure of the performance yet.

Third-party libraries used
--------------------------

- [ncurses][ncurses]
- [mattn/go-runewidth](https://github.com/mattn/go-runewidth)
    - Licensed under [MIT](http://mattn.mit-license.org/2013)
- [mattn/go-shellwords](https://github.com/mattn/go-shellwords)
    - Licensed under [MIT](http://mattn.mit-license.org/2014)

License
-------

[MIT](LICENSE)

[install]: https://github.com/junegunn/fzf#installation
[go]:      https://golang.org/
[gil]:     http://en.wikipedia.org/wiki/Global_Interpreter_Lock
[ncurses]: https://www.gnu.org/software/ncurses/
[req]:     http://golang.org/doc/install
[termbox]: https://github.com/nsf/termbox-go
