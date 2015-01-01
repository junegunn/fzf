fzf in Go
=========

This directory contains the source code for the new fzf implementation in Go.
This new version has the following benefits over the previous Ruby version.

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


Prebuilt binaries
-----------------

- Darwin x86_64
- Linux x86_64

Third-party libraries used
--------------------------

- [ncurses](https://www.gnu.org/software/ncurses/)
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

- [MIT](LICENSE)
