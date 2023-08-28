<img src="https://raw.githubusercontent.com/junegunn/i/master/fzf.png" height="170" alt="fzf - a command-line fuzzy finder"> [![github-actions](https://github.com/junegunn/fzf/workflows/Test%20fzf%20on%20Linux/badge.svg)](https://github.com/junegunn/fzf/actions)
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
    - Vim/Neovim plugin, key bindings, and fuzzy auto-completion

Sponsors ❤️
-----------

I would like to thank all the sponsors of this project who make it possible for me to continue to improve fzf.

If you'd like to sponsor this project, please visit https://github.com/sponsors/junegunn.

<!-- sponsors --><a href="https://github.com/miyanokomiya"><img src="https://github.com/miyanokomiya.png" width="60px" alt="miyanokomiya" /></a><a href="https://github.com/jonhoo"><img src="https://github.com/jonhoo.png" width="60px" alt="Jon Gjengset" /></a><a href="https://github.com/AceofSpades5757"><img src="https://github.com/AceofSpades5757.png" width="60px" alt="Kyle L. Davis" /></a><a href="https://github.com/Frederick888"><img src="https://github.com/Frederick888.png" width="60px" alt="Frederick Zhang" /></a><a href="https://github.com/moritzdietz"><img src="https://github.com/moritzdietz.png" width="60px" alt="Moritz Dietz" /></a><a href="https://github.com/mikker"><img src="https://github.com/mikker.png" width="60px" alt="Mikkel Malmberg" /></a><a href="https://github.com/dfsnow"><img src="https://github.com/dfsnow.png" width="60px" alt="Dan Snow" /></a><a href="https://github.com/pldubouilh"><img src="https://github.com/pldubouilh.png" width="60px" alt="Pierre Dubouilh" /></a><a href="https://github.com/rcorre"><img src="https://github.com/rcorre.png" width="60px" alt="Ryan Roden-Corrent" /></a><a href="https://github.com/getsentry"><img src="https://github.com/getsentry.png" width="60px" alt="Sentry" /></a><a href="https://github.com/blissdev"><img src="https://github.com/blissdev.png" width="60px" alt="Jordan Arentsen" /></a><a href="https://github.com/mislav"><img src="https://github.com/mislav.png" width="60px" alt="Mislav Marohnić" /></a><a href="https://github.com/aexvir"><img src="https://github.com/aexvir.png" width="60px" alt="Alex Viscreanu" /></a><a href="https://github.com/dbalatero"><img src="https://github.com/dbalatero.png" width="60px" alt="David Balatero" /></a><a href="https://github.com/comatory"><img src="https://github.com/comatory.png" width="60px" alt="Ondrej Synacek" /></a><a href="https://github.com/moobar"><img src="https://github.com/moobar.png" width="60px" alt="" /></a><a href="https://github.com/majjoha"><img src="https://github.com/majjoha.png" width="60px" alt="Mathias Jean Johansen" /></a><a href="https://github.com/benelan"><img src="https://github.com/benelan.png" width="60px" alt="Ben Elan" /></a><a href="https://github.com/jryom"><img src="https://github.com/jryom.png" width="60px" alt="Jesper" /></a><a href="https://github.com/nmrnv"><img src="https://github.com/nmrnv.png" width="60px" alt="Nikolay Marinov" /></a><a href="https://github.com/pawelduda"><img src="https://github.com/pawelduda.png" width="60px" alt="Paweł Duda" /></a><a href="https://github.com/slezica"><img src="https://github.com/slezica.png" width="60px" alt="Santiago Lezica" /></a><a href="https://github.com/pbwn"><img src="https://github.com/pbwn.png" width="60px" alt="" /></a><a href="https://github.com/timgluz"><img src="https://github.com/timgluz.png" width="60px" alt="Timo Sulg" /></a><a href="https://github.com/seanmorton"><img src="https://github.com/seanmorton.png" width="60px" alt="Sean Morton" /></a><a href="https://github.com/pyrho"><img src="https://github.com/pyrho.png" width="60px" alt="Damien Rajon" /></a><a href="https://github.com/ArtBIT"><img src="https://github.com/ArtBIT.png" width="60px" alt="ArtBIT" /></a><a href="https://github.com/da-moon"><img src="https://github.com/da-moon.png" width="60px" alt="" /></a><a href="https://github.com/hovissimo"><img src="https://github.com/hovissimo.png" width="60px" alt="Hovis" /></a><a href="https://github.com/dariusjonda"><img src="https://github.com/dariusjonda.png" width="60px" alt="Darius Jonda" /></a><a href="https://github.com/cristiand391"><img src="https://github.com/cristiand391.png" width="60px" alt="Cristian Dominguez" /></a><a href="https://github.com/eliangcs"><img src="https://github.com/eliangcs.png" width="60px" alt="Chang-Hung Liang" /></a><a href="https://github.com/raveensrk"><img src="https://github.com/raveensrk.png" width="60px" alt="Raveen Kumar" /></a><a href="https://github.com/asphaltbuffet"><img src="https://github.com/asphaltbuffet.png" width="60px" alt="Ben Lechlitner" /></a><a href="https://github.com/yash1th"><img src="https://github.com/yash1th.png" width="60px" alt="yash" /></a><!-- sponsors -->

Table of Contents
-----------------

<!-- vim-markdown-toc GFM -->

* [Installation](#installation)
    * [Using Homebrew](#using-homebrew)
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
* [`fzf-tmux` script](#fzf-tmux-script)
* [Key bindings for command-line](#key-bindings-for-command-line)
* [Fuzzy completion for bash and zsh](#fuzzy-completion-for-bash-and-zsh)
    * [Files and directories](#files-and-directories)
    * [Process IDs](#process-ids)
    * [Host names](#host-names)
    * [Environment variables / Aliases](#environment-variables--aliases)
    * [Settings](#settings)
    * [Supported commands](#supported-commands)
    * [Custom fuzzy completion](#custom-fuzzy-completion)
* [Vim plugin](#vim-plugin)
* [Advanced topics](#advanced-topics)
    * [Performance](#performance)
    * [Executing external programs](#executing-external-programs)
    * [Turning into a different process](#turning-into-a-different-process)
    * [Reloading the candidate list](#reloading-the-candidate-list)
        * [1. Update the list of processes by pressing CTRL-R](#1-update-the-list-of-processes-by-pressing-ctrl-r)
        * [2. Switch between sources by pressing CTRL-D or CTRL-F](#2-switch-between-sources-by-pressing-ctrl-d-or-ctrl-f)
        * [3. Interactive ripgrep integration](#3-interactive-ripgrep-integration)
    * [Preview window](#preview-window)
* [Tips](#tips)
    * [Respecting `.gitignore`](#respecting-gitignore)
    * [Fish shell](#fish-shell)
* [Related projects](#related-projects)
* [License](#license)

<!-- vim-markdown-toc -->

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

[bin]: https://github.com/junegunn/fzf/releases

### Using Homebrew

You can use [Homebrew](https://brew.sh/) (on macOS or Linux)
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

| Package Manager | Linux Distribution      | Command                            |
| ---             | ---                     | ---                                |
| APK             | Alpine Linux            | `sudo apk add fzf`                 |
| APT             | Debian 9+/Ubuntu 19.10+ | `sudo apt install fzf`             |
| Conda           |                         | `conda install -c conda-forge fzf` |
| DNF             | Fedora                  | `sudo dnf install fzf`             |
| Nix             | NixOS, etc.             | `nix-env -iA nixpkgs.fzf`          |
| Pacman          | Arch Linux              | `sudo pacman -S fzf`               |
| pkg             | FreeBSD                 | `pkg install fzf`                  |
| pkgin           | NetBSD                  | `pkgin install fzf`                |
| pkg_add         | OpenBSD                 | `pkg_add fzf`                      |
| Portage         | Gentoo                  | `emerge --ask app-shells/fzf`      |
| XBPS            | Void Linux              | `sudo xbps-install -S fzf`         |
| Zypper          | openSUSE                | `sudo zypper install fzf`          |

> :warning: **Key bindings (CTRL-T / CTRL-R / ALT-C) and fuzzy auto-completion
> may not be enabled by default.**
>
> Refer to the package documentation for more information. (e.g. `apt show fzf`)

[![Packaging status](https://repology.org/badge/vertical-allrepos/fzf.svg)](https://repology.org/project/fzf/versions)

### Windows

Pre-built binaries for Windows can be downloaded [here][bin]. fzf is also
available via [Chocolatey][choco], [Scoop][scoop], and [Winget][winget]:

| Package manager | Command              |
| ---             | ---                  |
| Chocolatey      | `choco install fzf`  |
| Scoop           | `scoop install fzf`  |
| Winget          | `winget install fzf` |

[choco]: https://chocolatey.org/packages/fzf
[scoop]: https://github.com/ScoopInstaller/Main/blob/master/bucket/fzf.json
[winget]: https://github.com/microsoft/winget-pkgs/tree/master/manifests/j/junegunn/fzf

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

fzf is being actively developed, and you might want to upgrade it once in a
while. Please follow the instruction below depending on the installation
method used.

- git: `cd ~/.fzf && git pull && ./install`
- brew: `brew update; brew upgrade fzf`
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

> *:bulb: A more robust solution would be to use `xargs` but we've presented
> the above as it's easier to grasp*
> ```sh
> fzf --print0 | xargs -0 -o vim
> ```

>
> *:bulb: fzf also has the ability to turn itself into a different process.*
>
> ```sh
> fzf --bind 'enter:become(vim {})'
> ```
>
> *See [Turning into a different process](#turning-into-a-different-process)
> for more information.*

### Using the finder

- `CTRL-K` / `CTRL-J` (or `CTRL-P` / `CTRL-N`) to move cursor up and down
- `Enter` key to select the item, `CTRL-C` / `CTRL-G` / `ESC` to exit
- On multi-select mode (`-m`), `TAB` and `Shift-TAB` to mark multiple items
- Emacs style key bindings
- Mouse: scroll, click, double-click; shift-click and shift-scroll on
  multi-select mode

### Layout

fzf by default starts in fullscreen mode, but you can make it start below the
cursor with `--height` option.

```sh
vim $(fzf --height 40%)
```

Also, check out `--reverse` and `--layout` options if you prefer
"top-down" layout instead of the default "bottom-up" layout.

```sh
vim $(fzf --height 40% --reverse)
```

You can add these options to `$FZF_DEFAULT_OPTS` so that they're applied by
default. For example,

```sh
export FZF_DEFAULT_OPTS='--height 40% --layout=reverse --border'
```

### Search syntax

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

### Environment variables

- `FZF_DEFAULT_COMMAND`
    - Default command to use when input is tty
    - e.g. `export FZF_DEFAULT_COMMAND='fd --type f'`
    - > :warning: This variable is not used by shell extensions due to the
      > slight difference in requirements.
      >
      > (e.g. `CTRL-T` runs `$FZF_CTRL_T_COMMAND` instead, `vim **<tab>` runs
      > `_fzf_compgen_path()`, and `cd **<tab>` runs `_fzf_compgen_dir()`)
      >
      > The available options are described later in this document.
- `FZF_DEFAULT_OPTS`
    - Default options
    - e.g. `export FZF_DEFAULT_OPTS="--layout=reverse --inline-info"`

### Options

See the man page (`man fzf`) for the full list of options.

### Demo
If you learn by watching videos, check out this screencast by [@samoshkin](https://github.com/samoshkin) to explore `fzf` features.

<a title="fzf - command-line fuzzy finder" href="https://www.youtube.com/watch?v=qgG5Jhi_Els">
  <img src="https://i.imgur.com/vtG8olE.png" width="640">
</a>

Examples
--------

* [Wiki page of examples](https://github.com/junegunn/fzf/wiki/examples)
    * *Disclaimer: The examples on this page are maintained by the community
      and are not thoroughly tested*
* [Advanced fzf examples](https://github.com/junegunn/fzf/blob/master/ADVANCED.md)

`fzf-tmux` script
-----------------

[fzf-tmux](bin/fzf-tmux) is a bash script that opens fzf in a tmux pane.

```sh
# usage: fzf-tmux [LAYOUT OPTIONS] [--] [FZF OPTIONS]

# See available options
fzf-tmux --help

# select git branches in horizontal split below (15 lines)
git branch | fzf-tmux -d 15

# select multiple words in vertical split on the left (20% of screen width)
cat /usr/share/dict/words | fzf-tmux -l 20% --multi --reverse
```

It will still work even when you're not on tmux, silently ignoring `-[pudlr]`
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
    - Set `FZF_CTRL_T_OPTS` to pass additional options to fzf
      ```sh
      # Preview file content using bat (https://github.com/sharkdp/bat)
      export FZF_CTRL_T_OPTS="
        --preview 'bat -n --color=always {}'
        --bind 'ctrl-/:change-preview-window(down|hidden|)'"
      ```
- `CTRL-R` - Paste the selected command from history onto the command-line
    - If you want to see the commands in chronological order, press `CTRL-R`
      again which toggles sorting by relevance
    - Set `FZF_CTRL_R_OPTS` to pass additional options to fzf
      ```sh
      # CTRL-/ to toggle small preview window to see the full command
      # CTRL-Y to copy the command into clipboard using pbcopy
      export FZF_CTRL_R_OPTS="
        --preview 'echo {}' --preview-window up:3:hidden:wrap
        --bind 'ctrl-/:toggle-preview'
        --bind 'ctrl-y:execute-silent(echo -n {2..} | pbcopy)+abort'
        --color header:italic
        --header 'Press CTRL-Y to copy command into clipboard'"
      ```
- `ALT-C` - cd into the selected directory
    - Set `FZF_ALT_C_COMMAND` to override the default command
    - Set `FZF_ALT_C_OPTS` to pass additional options to fzf
      ```sh
      # Print tree structure in the preview window
      export FZF_ALT_C_OPTS="--preview 'tree -C {}'"
      ```

If you're on a tmux session, you can start fzf in a tmux split-pane or in
a tmux popup window by setting `FZF_TMUX_OPTS` (e.g. `export FZF_TMUX_OPTS='-p80%,60%'`).
See `fzf-tmux --help` for available options.

More tips can be found on [the wiki page](https://github.com/junegunn/fzf/wiki/Configuring-shell-key-bindings).

Fuzzy completion for bash and zsh
---------------------------------

### Files and directories

Fuzzy completion for files and directories can be triggered if the word before
the cursor ends with the trigger sequence, which is by default `**`.

- `COMMAND [DIRECTORY/][FUZZY_PATTERN]**<TAB>`

```sh
# Files under the current directory
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

### Process IDs

Fuzzy completion for PIDs is provided for kill command.

```sh
# Can select multiple processes with <TAB> or <Shift-TAB> keys
kill -9 **<TAB>
```

### Host names

For ssh and telnet commands, fuzzy completion for hostnames is provided. The
names are extracted from /etc/hosts and ~/.ssh/config.

```sh
ssh **<TAB>
telnet **<TAB>
```

### Environment variables / Aliases

```sh
unset **<TAB>
export **<TAB>
unalias **<TAB>
```

### Settings

```sh
# Use ~~ as the trigger sequence instead of the default **
export FZF_COMPLETION_TRIGGER='~~'

# Options to fzf command
export FZF_COMPLETION_OPTS='--border --info=inline'

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

# Advanced customization of fzf options via _fzf_comprun function
# - The first argument to the function is the name of the command.
# - You should make sure to pass the rest of the arguments to fzf.
_fzf_comprun() {
  local command=$1
  shift

  case "$command" in
    cd)           fzf --preview 'tree -C {} | head -200'   "$@" ;;
    export|unset) fzf --preview "eval 'echo \$'{}"         "$@" ;;
    ssh)          fzf --preview 'dig {}'                   "$@" ;;
    *)            fzf --preview 'bat -n --color=always {}' "$@" ;;
  esac
}
```

### Supported commands

On bash, fuzzy completion is enabled only for a predefined set of commands
(`complete | grep _fzf` to see the list). But you can enable it for other
commands as well by using `_fzf_setup_completion` helper function.

```sh
# usage: _fzf_setup_completion path|dir|var|alias|host COMMANDS...
_fzf_setup_completion path ag git kubectl
_fzf_setup_completion dir tree
```

### Custom fuzzy completion

_**(Custom completion API is experimental and subject to change)**_

For a command named _"COMMAND"_, define `_fzf_complete_COMMAND` function using
`_fzf_complete` helper.

```sh
# Custom fuzzy completion for "doge" command
#   e.g. doge **<TAB>
_fzf_complete_doge() {
  _fzf_complete --multi --reverse --prompt="doge> " -- "$@" < <(
    echo very
    echo wow
    echo such
    echo doge
  )
}
```

- The arguments before `--` are the options to fzf.
- After `--`, simply pass the original completion arguments unchanged (`"$@"`).
- Then, write a set of commands that generates the completion candidates and
  feed its output to the function using process substitution (`< <(...)`).

zsh will automatically pick up the function using the naming convention but in
bash you have to manually associate the function with the command using the
`complete` command.

```sh
[ -n "$BASH" ] && complete -F _fzf_complete_doge -o default -o bashdefault doge
```

If you need to post-process the output from fzf, define
`_fzf_complete_COMMAND_post` as follows.

```sh
_fzf_complete_foo() {
  _fzf_complete --multi --reverse --header-lines=3 -- "$@" < <(
    ls -al
  )
}

_fzf_complete_foo_post() {
  awk '{print $NF}'
}

[ -n "$BASH" ] && complete -F _fzf_complete_foo -o default -o bashdefault foo
```

Vim plugin
----------

See [README-VIM.md](README-VIM.md).

Advanced topics
---------------

### Performance

fzf is fast. Performance should not be a problem in most use cases. However,
you might want to be aware of the options that can affect performance.

- `--ansi` tells fzf to extract and parse ANSI color codes in the input, and it
  makes the initial scanning slower. So it's not recommended that you add it
  to your `$FZF_DEFAULT_OPTS`.
- `--nth` makes fzf slower because it has to tokenize each line.
- `--with-nth` makes fzf slower as fzf has to tokenize and reassemble each
  line.

### Executing external programs

You can set up key bindings for starting external processes without leaving
fzf (`execute`, `execute-silent`).

```bash
# Press F1 to open the file with less without leaving fzf
# Press CTRL-Y to copy the line to clipboard and aborts fzf (requires pbcopy)
fzf --bind 'f1:execute(less -f {}),ctrl-y:execute-silent(echo {} | pbcopy)+abort'
```

See *KEY BINDINGS* section of the man page for details.

### Turning into a different process

`become(...)` is similar to `execute(...)`/`execute-silent(...)` described
above, but instead of executing the command and coming back to fzf on
complete, it turns fzf into a new process for the command.

```sh
fzf --bind 'enter:become(vim {})'
```

Compared to the seemingly equivalent command substitution `vim "$(fzf)"`, this
approach has several advantages:

* Vim will not open an empty file when you terminate fzf with
  <kbd>CTRL-C</kbd>
* Vim will not open an empty file when you press <kbd>ENTER</kbd> on an empty
  result
* Can handle multiple selections even when they have whitespaces
  ```sh
  fzf --multi --bind 'enter:become(vim {+})'
  ```

To be fair, running `fzf --print0 | xargs -0 -o vim` instead of `vim "$(fzf)"`
resolves all of the issues mentioned. Nonetheless, `become(...)` still offers
additional benefits in different scenarios.

* You can set up multiple bindings to handle the result in different ways
  without any wrapping script
  ```sh
  fzf --bind 'enter:become(vim {}),ctrl-e:become(emacs {})'
  ```
  * Previously, you would have to use `--expect=ctrl-e` and check the first
    line of the output of fzf
* You can easily build the subsequent command using the field index
  expressions of fzf
  ```sh
  # Open the file in Vim and go to the line
  git grep --line-number . |
      fzf --delimiter : --nth 3.. --bind 'enter:become(vim {1} +{2})'
  ```

### Reloading the candidate list

By binding `reload` action to a key or an event, you can make fzf dynamically
reload the candidate list. See https://github.com/junegunn/fzf/issues/1750 for
more details.

#### 1. Update the list of processes by pressing CTRL-R

```sh
ps -ef |
  fzf --bind 'ctrl-r:reload(ps -ef)' \
      --header 'Press CTRL-R to reload' --header-lines=1 \
      --height=50% --layout=reverse
```

#### 2. Switch between sources by pressing CTRL-D or CTRL-F

```sh
FZF_DEFAULT_COMMAND='find . -type f' \
  fzf --bind 'ctrl-d:reload(find . -type d),ctrl-f:reload(eval "$FZF_DEFAULT_COMMAND")' \
      --height=50% --layout=reverse
```

#### 3. Interactive ripgrep integration

The following example uses fzf as the selector interface for ripgrep. We bound
`reload` action to `change` event, so every time you type on fzf, the ripgrep
process will restart with the updated query string denoted by the placeholder
expression `{q}`. Also, note that we used `--disabled` option so that fzf
doesn't perform any secondary filtering.

```sh
: | rg_prefix='rg --column --line-number --no-heading --color=always --smart-case' \
    fzf --bind 'start:reload:$rg_prefix ""' \
        --bind 'change:reload:$rg_prefix {q} || true' \
        --bind 'enter:become(vim {1} +{2})' \
        --ansi --disabled \
        --height=50% --layout=reverse
```

If ripgrep doesn't find any matches, it will exit with a non-zero exit status,
and fzf will warn you about it. To suppress the warning message, we added
`|| true` to the command, so that it always exits with 0.

See ["Using fzf as interactive Ripgrep launcher"](https://github.com/junegunn/fzf/blob/master/ADVANCED.md#using-fzf-as-interactive-ripgrep-launcher)
for more sophisticated examples.

### Preview window

When the `--preview` option is set, fzf automatically starts an external process
with the current line as the argument and shows the result in the split window.
Your `$SHELL` is used to execute the command with `$SHELL -c COMMAND`.
The window can be scrolled using the mouse or custom key bindings.

```bash
# {} is replaced with the single-quoted string of the focused line
fzf --preview 'cat {}'
```

Preview window supports ANSI colors, so you can use any program that
syntax-highlights the content of a file, such as
[Bat](https://github.com/sharkdp/bat) or
[Highlight](http://www.andre-simon.de/doku/highlight/en/highlight.php):

```bash
fzf --preview 'bat --color=always {}' --preview-window '~3'
```

You can customize the size, position, and border of the preview window using
`--preview-window` option, and the foreground and background color of it with
`--color` option. For example,

```bash
fzf --height 40% --layout reverse --info inline --border \
    --preview 'file {}' --preview-window up,1,border-horizontal \
    --bind 'ctrl-/:change-preview-window(50%|hidden|)' \
    --color 'fg:#bbccdd,fg+:#ddeeff,bg:#334455,preview-bg:#223344,border:#778899'
```

See the man page (`man fzf`) for the full list of options.

More advanced examples can be found [here](https://github.com/junegunn/fzf/blob/master/ADVANCED.md).

----

Since fzf is a general-purpose text filter rather than a file finder, **it is
not a good idea to add `--preview` option to your `$FZF_DEFAULT_OPTS`**.

```sh
# *********************
# ** DO NOT DO THIS! **
# *********************
export FZF_DEFAULT_OPTS='--preview "bat --style=numbers --color=always --line-range :500 {}"'

# bat doesn't work with any input other than the list of files
ps -ef | fzf
seq 100 | fzf
history | fzf
```

Tips
----

### Respecting `.gitignore`

You can use [fd](https://github.com/sharkdp/fd),
[ripgrep](https://github.com/BurntSushi/ripgrep), or [the silver
searcher](https://github.com/ggreer/the_silver_searcher) instead of the
default find command to traverse the file system while respecting
`.gitignore`.

```sh
# Feed the output of fd into fzf
fd --type f --strip-cwd-prefix | fzf

# Setting fd as the default source for fzf
export FZF_DEFAULT_COMMAND='fd --type f --strip-cwd-prefix'

# Now fzf (w/o pipe) will use fd instead of find
fzf

# To apply the command to CTRL-T as well
export FZF_CTRL_T_COMMAND="$FZF_DEFAULT_COMMAND"
```

If you want the command to follow symbolic links and don't want it to exclude
hidden files, use the following command:

```sh
export FZF_DEFAULT_COMMAND='fd --type f --strip-cwd-prefix --hidden --follow --exclude .git'
```

### Fish shell

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

Copyright (c) 2013-2023 Junegunn Choi
