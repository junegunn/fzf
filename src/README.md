fzf in Go
=========

This directory contains the source code for the new fzf implementation in
[Go][go]. This new version has the following benefits over the previous Ruby
version.

- Immensely faster
    - No GIL. Performance is linearly proportional to the number of cores.
    - It's so fast that I even decided to remove the sort limit (`--sort=N`)
- Does not require Ruby and distributed as an executable binary
    - Ruby dependency is especially painful on Ruby 2.1 or above which
      ships without curses gem

Build
-----

```sh
# Build fzf executable
make

# Install the executable to ../bin directory
make install

# Build executable for Linux x86_64 using Docker
make linux64
```

System requirements
-------------------

Currently prebuilt binaries are provided only for 64 bit OS X and Linux.
The install script will fall back to the legacy Ruby version on the other
systems, but if you have Go installed, you can try building it yourself.
(`make install`)

However, as pointed out in [golang.org/doc/install][req], the Go version will
not run on CentOS/RHEL 5.x and thus the install script will choose the Ruby
version instead.

The Go version depends on [ncurses][ncurses] and some Unix system calls, so it
shouldn't run natively on Windows at the moment. But it should be not
impossible to support Windows by falling back to a cross-platform alternative
such as [termbox][termbox] only on Windows. If you're interested in making fzf
work on Windows, please let me know.

Third-party libraries used
--------------------------

- [ncurses][ncurses]
- [mattn/go-runewidth](https://github.com/mattn/go-runewidth)
    - Licensed under [MIT](http://mattn.mit-license.org/2013)
- [mattn/go-shellwords](https://github.com/mattn/go-shellwords)
    - Licensed under [MIT](http://mattn.mit-license.org/2014)

Contribution
------------

For the moment, I will not add or accept any new features until we can be sure
that the implementation is stable and we have a sufficient number of test
cases. However, fixes for obvious bugs and new test cases are welcome.

I also care much about the performance of the implementation (that's the
reason I rewrote the whole thing in Go, right?), so please make sure that your
change does not result in performance regression. Please be minded that we
still don't have a quantitative measure of the performance.

License
-------

[MIT](LICENSE)

[go]:      https://golang.org/
[ncurses]: https://www.gnu.org/software/ncurses/
[req]:     http://golang.org/doc/install
[termbox]: https://github.com/nsf/termbox-go
