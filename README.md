<img src="https://raw.githubusercontent.com/junegunn/i/master/fzf.png" height="170" alt="fzf - a command-line fuzzy finder"> [![travis-ci](https://travis-ci.org/junegunn/fzf.svg?branch=master)](https://travis-ci.org/junegunn/fzf)
===

fzf is a general-purpose command-line fuzzy finder.

<img src="https://raw.githubusercontent.com/junegunn/i/master/fzf-preview.png" width=640>

It's an interactive Unix filter for command-line that can be used with any
list; files, command history, processes, hostnames, bookmarks, git commits,
etc.

Pros
----

- Portable, no dependencies
- Blazingly fast
- The most comprehensive feature set
- Flexible layout
- Batteries included
    - Vim/Neovim plugin, key bindings and fuzzy auto-completion

Table of Contents
-----------------

   * [Installation](#installation)
      * [Using Homebrew or Linuxbrew](#using-homebrew-or-linuxbrew)
      * [Using git](#using-git)
      * [Using Linux package managers](#using-linux-package-managers)
      * [Windows](#windows)
      * [As Vim plugin](#as-vim-plugin)
   * [Upgrading fzf](#upgrading-fzf)
   * [Building fzf](#building-fzf)
   * [Usage](#usage)
      * [Using the finder](#using-the-finder)
      * [Layout](#layout)
      * [Search syntax](#search-syntax)
      * [Environment variables](#environment-variables)
      * [Options](#options)
      * [Demo](#demo)
   * [Examples](#examples)
   * [fzf-tmux script](#fzf-tmux-script)
   * [Key bindings for command line](#key-bindings-for-command-line)
   * [Fuzzy completion for bash and zsh](#fuzzy-completion-for-bash-and-zsh)
      * [Files and directories](#files-and-directories)
      * [Process IDs](#process-ids)
      * [Host names](#host-names)
      * [Environment variables / Aliases](#environment-variables--aliases)
      * [Settings](#settings)
      * [Supported commands](#supported-commands)
   * [Vim plugin](#vim-plugin)
   * [Advanced topics](#advanced-topics)
      * [Performance](#performance)
      * [Executing external programs](#executing-external-programs)
      * [Preview window](#preview-window)
   * [Tips](#tips)
      * [Respecting .gitignore](#respecting-gitignore)
      * [Fish shell](#fish-shell)
   * [Related projects](#related-projects)
   * [<a href="LICENSE">License</a>](#license)

Installation
------------

fzf project consists of the following components:

- `fzf` executable
- `fzf-tmux` script for launching fzf in a tmux pane
- Shell extensions
    - Key bindings (`CTRL-T`, `CTRL-R`, and `ALT-C`) (bash, zsh, fish)
    - Fuzzy auto-completion (bash, zsh)
- Vim/Neovim plugin

You can [download fzf executable][bin] alone if you don't need the extra
stuff.

[bin]: https://github.com/junegunn/fzf-bin/releases

### Using Homebrew or Linuxbrew

You can use [Homebrew](http://brew.sh/) or [Linuxbrew](http://linuxbrew.sh/)
to install fzf.

```sh
brew install fzf

# To install useful key bindings and fuzzy completion:
$(brew --prefix)/opt/fzf/install
```

fzf is also available [via MacPorts][portfile]: `sudo port install fzf`

[portfile]: https://github.com/macports/macports-ports/blob/master/sysutils/fzf/Portfile

### Using git

Alternatively, you can "git clone" this repository to any directory and run
[install](https://github.com/junegunn/fzf/blob/master/install) script.

```sh
git clone --depth 1 https://github.com/junegunn/fzf.git ~/.fzf
~/.fzf/install
```

### Using Linux package managers

| Distro       | Command                    |
| ---          | ---                        |
| Alpine Linux | `sudo apk add fzf`         |
| Arch Linux   | `sudo pacman -S fzf`       |
| Debian       | `sudo apt-get install fzf` |
| Fedora       | `sudo dnf install fzf`     |
| FreeBSD      | `pkg install fzf`          |
| NixOS        | `nix-env -iA nixpkgs.fzf`  |
| openSUSE     | `sudo zypper install fzf`  |
| OpenBSD      | `pkg_add fzf`              |

Shell extensions (key bindings and fuzzy auto-completion) and Vim/Neovim
plugin may or may not be enabled by default depending on the package manager.
Refer to the package documentation for more information.

### Windows

Pre-built binaries for Windows can be downloaded [here][bin]. fzf is also
available via [Chocolatey][choco] and [Scoop][scoop]:

| Package manager | Command             |
| ---             | ---                 |
| Chocolatey      | `choco install fzf` |
| Scoop           | `scoop install fzf` |

[choco]: https://chocolatey.org/packages/fzf
[scoop]: https://github.com/ScoopInstaller/Main/blob/master/bucket/fzf.json

Known issues and limitations on Windows can be found on [the wiki
page][windows-wiki].

[windows-wiki]: https://github.com/junegunn/fzf/wiki/Windows

### As Vim plugin

If you use
[vim-plug](https://github.com/junegunn/vim-plug), add this line to your Vim
configuration file:

```vim
Plug 'junegunn/fzf', { 'do': { -> fzf#install() } }
```

`fzf#install()` makes sure that you have the latest binary, but it's optional,
so you can omit it if you use a plugin manager that doesn't support hooks.

For more installation options, see [README-VIM.md](README-VIM.md).

Upgrading fzf
-------------

fzf is being actively developed and you might want to upgrade it once in a
while. Please follow the instruction below depending on the installation
method used.

- git: `cd ~/.fzf && git pull && ./install`
- brew: `brew update; brew reinstall fzf`
- macports: `sudo port upgrade fzf`
- chocolatey: `choco upgrade fzf`
- vim-plug: `:PlugUpdate fzf`

Building fzf
------------

See [BUILD.md](BUILD.md).

Usage
-----

fzf will launch interactive finder, read the list from STDIN, and write the
selected item to STDOUT.

```sh
find * -type f | fzf > selected
```

Without STDIN pipe, fzf will use find command to fetch the list of
files excluding hidden ones. (You can override the default command with
`FZF_DEFAULT_COMMAND`)

```sh
vim $(fzf)
```

#### Using the finder

- `CTRL-J` / `CTRL-K` (or `CTRL-N` / `CTRL-P`) to move cursor up and down
- `Enter` key to select the item, `CTRL-C` / `CTRL-G` / `ESC` to exit
- On multi-select mode (`-m`), `TAB` and `Shift-TAB` to mark multiple items
- Emacs style key bindings
- Mouse: scroll, click, double-click; shift-click and shift-scroll on
  multi-select mode

#### Layout

fzf by default starts in fullscreen mode, but you can make it start below the
cursor with `--height` option.

```sh
vim $(fzf --height 40%)
```

Also check out `--reverse` and `--layout` options if you prefer
"top-down" layout instead of the default "bottom-up" layout.

```sh
vim $(fzf --height 40% --reverse)
```

You can add these options to `$FZF_DEFAULT_OPTS` so that they're applied by
default. For example,

```sh
export FZF_DEFAULT_OPTS='--height 40% --layout=reverse --border'
```

#### Search syntax

Unless otherwise specified, fzf starts in "extended-search mode" where you can
type in multiple search terms delimited by spaces. e.g. `^music .mp3$ sbtrkt
!fire`

| Token     | Match type                 | Description                          |
| --------- | -------------------------- | ------------------------------------ |
| `sbtrkt`  | fuzzy-match                | Items that match `sbtrkt`            |
| `'wild`   | exact-match (quoted)       | Items that include `wild`            |
| `^music`  | prefix-exact-match         | Items that start with `music`        |
| `.mp3$`   | suffix-exact-match         | Items that end with `.mp3`           |
| `!fire`   | inverse-exact-match        | Items that do not include `fire`     |
| `!^music` | inverse-prefix-exact-match | Items that do not start with `music` |
| `!.mp3$`  | inverse-suffix-exact-match | Items that do not end with `.mp3`    |

If you don't prefer fuzzy matching and do not wish to "quote" every word,
start fzf with `-e` or `--exact` option. Note that when  `--exact` is set,
`'`-prefix "unquotes" the term.

A single bar character term acts as an OR operator. For example, the following
query matches entries that start with `core` and end with either `go`, `rb`,
or `py`.

```
^core go$ | rb$ | py$
```

#### Environment variables

- `FZF_DEFAULT_COMMAND`
    - Default command to use when input is tty
    - e.g. `export FZF_DEFAULT_COMMAND='fd --type f'`
- `FZF_DEFAULT_OPTS`
    - Default options
    - e.g. `export FZF_DEFAULT_OPTS="--layout=reverse --inline-info"`

#### Options

See the man page (`man fzf`) for the full list of options.

#### Demo
If you learn by watching videos, check out this screencast by [@samoshkin](https://github.com/samoshkin) to explore `fzf` features.

<a title="fzf - command-line fuzzy finder" href="https://www.youtube.com/watch?v=qgG5Jhi_Els">
  <img src="https://i.imgur.com/vtG8olE.png" width="640">
</a>

Examples
--------

Many useful examples can be found on [the wiki
page](https://github.com/junegunn/fzf/wiki/examples). Feel free to add your
own as well.

`fzf-tmux` script
-----------------

[fzf-tmux](bin/fzf-tmux) is a bash script that opens fzf in a tmux pane.

```sh
# usage: fzf-tmux [-u|-d [HEIGHT[%]]] [-l|-r [WIDTH[%]]] [--] [FZF OPTIONS]
#        (-[udlr]: up/down/left/right)

# select git branches in horizontal split below (15 lines)
git branch | fzf-tmux -d 15

# select multiple words in vertical split on the left (20% of screen width)
cat /usr/share/dict/words | fzf-tmux -l 20% --multi --reverse
```

It will still work even when you're not on tmux, silently ignoring `-[udlr]`
options, so you can invariably use `fzf-tmux` in your scripts.

Alternatively, you can use `--height HEIGHT[%]` option not to start fzf in
fullscreen mode.

```sh
fzf --height 40%
```

Key bindings for command-line
-----------------------------

The install script will setup the following key bindings for bash, zsh, and
fish.

- `CTRL-T` - Paste the selected files and directories onto the command-line
    - Set `FZF_CTRL_T_COMMAND` to override the default command
    - Set `FZF_CTRL_T_OPTS` to pass additional options
- `CTRL-R` - Paste the selected command from history onto the command-line
    - If you want to see the commands in chronological order, press `CTRL-R`
      again which toggles sorting by relevance
    - Set `FZF_CTRL_R_OPTS` to pass additional options
- `ALT-C` - cd into the selected directory
    - Set `FZF_ALT_C_COMMAND` to override the default command
    - Set `FZF_ALT_C_OPTS` to pass additional options

If you're on a tmux session, you can start fzf in a split pane by setting
`FZF_TMUX` to 1, and change the height of the pane with `FZF_TMUX_HEIGHT`
(e.g. `20`, `50%`).

More tips can be found on [the wiki page](https://github.com/junegunn/fzf/wiki/Configuring-shell-key-bindings).

Fuzzy completion for bash and zsh
---------------------------------

#### Files and directories

Fuzzy completion for files and directories can be triggered if the word before
the cursor ends with the trigger sequence which is by default `**`.

- `COMMAND [DIRECTORY/][FUZZY_PATTERN]**<TAB>`

```sh
# Files under current directory
# - You can select multiple items with TAB key
vim **<TAB>

# Files under parent directory
vim ../**<TAB>

# Files under parent directory that match `fzf`
vim ../fzf**<TAB>

# Files under your home directory
vim ~/**<TAB>


# Directories under current directory (single-selection)
cd **<TAB>

# Directories under ~/github that match `fzf`
cd ~/github/fzf**<TAB>
```

#### Process IDs

Fuzzy completion for PIDs is provided for kill command. In this case,
there is no trigger sequence, just press tab key after kill command.

```sh
# Can select multiple processes with <TAB> or <Shift-TAB> keys
kill -9 <TAB>
```

#### Host names

For ssh and telnet commands, fuzzy completion for host names is provided. The
names are extracted from /etc/hosts and ~/.ssh/config.

```sh
ssh **<TAB>
telnet **<TAB>
```

#### Environment variables / Aliases

```sh
unset **<TAB>
export **<TAB>
unalias **<TAB>
```

#### Settings

```sh
# Use ~~ as the trigger sequence instead of the default **
export FZF_COMPLETION_TRIGGER='~~'

# Options to fzf command
export FZF_COMPLETION_OPTS='+c -x'

# Use fd (https://github.com/sharkdp/fd) instead of the default find
# command for listing path candidates.
# - The first argument to the function ($1) is the base path to start traversal
# - See the source code (completion.{bash,zsh}) for the details.
_fzf_compgen_path() {
  fd --hidden --follow --exclude ".git" . "$1"
}

# Use fd to generate the list for directory completion
_fzf_compgen_dir() {
  fd --type d --hidden --follow --exclude ".git" . "$1"
}

# (EXPERIMENTAL) Advanced customization of fzf options via _fzf_comprun function
# - The first argument to the function is the name of the command.
# - You should make sure to pass the rest of the arguments to fzf.
_fzf_comprun() {
  local command=$1
  shift

  case "$command" in
    cd)           fzf "$@" --preview 'tree -C {} | head -200' ;;
    export|unset) fzf "$@" --preview "eval 'echo \$'{}" "$@" ;;
    ssh)          fzf "$@" --preview 'dig {}' ;;
    *)            fzf "$@" ;;
  esac
}
```

#### Supported commands

On bash, fuzzy completion is enabled only for a predefined set of commands
(`complete | grep _fzf` to see the list). But you can enable it for other
commands as well by using `_fzf_setup_completion` helper function.

```sh
# usage: _fzf_setup_completion path|dir|var|alias|host COMMANDS...
_fzf_setup_completion path ag git kubectl
_fzf_setup_completion dir tree
```

Vim plugin
----------

See [README-VIM.md](README-VIM.md).

Advanced topics
---------------

### Performance

fzf is fast and is [getting even faster][perf]. Performance should not be
a problem in most use cases. However, you might want to be aware of the
options that affect the performance.

- `--ansi` tells fzf to extract and parse ANSI color codes in the input and it
  makes the initial scanning slower. So it's not recommended that you add it
  to your `$FZF_DEFAULT_OPTS`.
- `--nth` makes fzf slower as fzf has to tokenize each line.
- `--with-nth` makes fzf slower as fzf has to tokenize and reassemble each
  line.
- If you absolutely need better performance, you can consider using
  `--algo=v1` (the default being `v2`) to make fzf use a faster greedy
  algorithm. However, this algorithm is not guaranteed to find the optimal
  ordering of the matches and is not recommended.

[perf]: https://junegunn.kr/images/fzf-0.17.0.png

### Executing external programs

You can set up key bindings for starting external processes without leaving
fzf (`execute`, `execute-silent`).

```bash
# Press F1 to open the file with less without leaving fzf
# Press CTRL-Y to copy the line to clipboard and aborts fzf (requires pbcopy)
fzf --bind 'f1:execute(less -f {}),ctrl-y:execute-silent(echo {} | pbcopy)+abort'
```

See *KEY BINDINGS* section of the man page for details.

### Preview window

When `--preview` option is set, fzf automatically starts an external process with
the current line as the argument and shows the result in the split window. Your
`$SHELL` is used to execute the command with `$SHELL -c COMMAND`.

```bash
# {} is replaced to the single-quoted string of the focused line
fzf --preview 'cat {}'
```

Since the preview window is updated only after the process is complete, it's
important that the command finishes quickly.

```bash
# Use head instead of cat so that the command doesn't take too long to finish
fzf --preview 'head -100 {}'
```

Preview window supports ANSI colors, so you can use any program that
syntax-highlights the content of a file.

- Bat: https://github.com/sharkdp/bat
- Highlight: http://www.andre-simon.de/doku/highlight/en/highlight.php

```bash
fzf --preview 'bat --style=numbers --color=always {} | head -500'
```

You can customize the size, position, and border of the preview window using
`--preview-window` option, and the foreground and background color of it with
`--color` option. For example,

```bash
fzf --height 40% --layout reverse --info inline --border \
    --preview 'file {}' --preview-window down:1:noborder \
    --color 'fg:#bbccdd,fg+:#ddeeff,bg:#334455,preview-bg:#223344,border:#778899'
```

See the man page (`man fzf`) for the full list of options.

For more advanced examples, see [Key bindings for git with fzf][fzf-git]
([code](https://gist.github.com/junegunn/8b572b8d4b5eddd8b85e5f4d40f17236)).

[fzf-git]: https://junegunn.kr/2016/07/fzf-git/

----

Since fzf is a general-purpose text filter rather than a file finder, **it is
not a good idea to add `--preview` option to your `$FZF_DEFAULT_OPTS`**.

```sh
# *********************
# ** DO NOT DO THIS! **
# *********************
export FZF_DEFAULT_OPTS='--preview "bat --style=numbers --color=always {} | head -500"'

# bat doesn't work with any input other than the list of files
ps -ef | fzf
seq 100 | fzf
history | fzf
```

Tips
----

#### Respecting `.gitignore`

You can use [fd](https://github.com/sharkdp/fd),
[ripgrep](https://github.com/BurntSushi/ripgrep), or [the silver
searcher](https://github.com/ggreer/the_silver_searcher) instead of the
default find command to traverse the file system while respecting
`.gitignore`.

```sh
# Feed the output of fd into fzf
fd --type f | fzf

# Setting fd as the default source for fzf
export FZF_DEFAULT_COMMAND='fd --type f'

# Now fzf (w/o pipe) will use fd instead of find
fzf

# To apply the command to CTRL-T as well
export FZF_CTRL_T_COMMAND="$FZF_DEFAULT_COMMAND"
```

If you want the command to follow symbolic links, and don't want it to exclude
hidden files, use the following command:

```sh
export FZF_DEFAULT_COMMAND='fd --type f --hidden --follow --exclude .git'
```

#### Fish shell

`CTRL-T` key binding of fish, unlike those of bash and zsh, will use the last
token on the command-line as the root directory for the recursive search. For
instance, hitting `CTRL-T` at the end of the following command-line

```sh
ls /var/
```

will list all files and directories under `/var/`.

When using a custom `FZF_CTRL_T_COMMAND`, use the unexpanded `$dir` variable to
make use of this feature. `$dir` defaults to `.` when the last token is not a
valid directory. Example:

```sh
set -g FZF_CTRL_T_COMMAND "command find -L \$dir -type f 2> /dev/null | sed '1d; s#^\./##'"
```

Related projects
----------------

https://github.com/junegunn/fzf/wiki/Related-projects

[License](LICENSE)
------------------

The MIT License (MIT)

Copyright (c) 2013-2020 Junegunn Choi
