<div align="center">
<sup>Special thanks to:</sup>
<br>
<br>
<a href="https://warp.dev/?utm_source=github&utm_medium=referral&utm_campaign=fzf_20240209">
  <div>
    <img src="https://raw.githubusercontent.com/junegunn/i/master/warp.png" width="300" alt="Warp">
  </div>
  <b>Warp is a modern, Rust-based terminal with AI built in so you and your team can build great software, faster.</b>
  <div>
    <sup>Visit warp.dev to learn more.</sup>
  </div>
</a>
<br>
<hr>
</div>
<br>

<img src="https://raw.githubusercontent.com/junegunn/i/master/fzf.png" height="170" alt="fzf - a command-line fuzzy finder"> [![github-actions](https://github.com/junegunn/fzf/workflows/Test%20fzf%20on%20Linux/badge.svg)](https://github.com/junegunn/fzf/actions)
===

fzf is a general-purpose command-line fuzzy finder.

<img src="https://raw.githubusercontent.com/junegunn/i/master/fzf-preview.png" width=640>

It's an interactive filter program for any kind of list; files, command
history, processes, hostnames, bookmarks, git commits, etc. It implements
a "fuzzy" matching algorithm, so you can quickly type in patterns with omitted
characters and still get the results you want.

Pros
----

- Portable, no dependencies
- Blazingly fast
- Extremely versatile
- Batteries included
    - bash/zsh/fish integration, tmux integration, Vim/Neovim plugin

Sponsors ❤️
-----------

I would like to thank all the sponsors of this project who make it possible for me to continue to improve fzf.

If you'd like to sponsor this project, please visit https://github.com/sponsors/junegunn.

<!-- sponsors --><a href="https://github.com/miyanokomiya"><img src="https://github.com/miyanokomiya.png" width="60px" alt="miyanokomiya" /></a><a href="https://github.com/jonhoo"><img src="https://github.com/jonhoo.png" width="60px" alt="Jon Gjengset" /></a><a href="https://github.com/AceofSpades5757"><img src="https://github.com/AceofSpades5757.png" width="60px" alt="Kyle L. Davis" /></a><a href="https://github.com/Frederick888"><img src="https://github.com/Frederick888.png" width="60px" alt="Frederick Zhang" /></a><a href="https://github.com/moritzdietz"><img src="https://github.com/moritzdietz.png" width="60px" alt="Moritz Dietz" /></a><a href="https://github.com/mikker"><img src="https://github.com/mikker.png" width="60px" alt="Mikkel Malmberg" /></a><a href="https://github.com/pldubouilh"><img src="https://github.com/pldubouilh.png" width="60px" alt="Pierre Dubouilh" /></a><a href="https://github.com/trantor"><img src="https://github.com/trantor.png" width="60px" alt="Fulvio Scapin" /></a><a href="https://github.com/rcorre"><img src="https://github.com/rcorre.png" width="60px" alt="Ryan Roden-Corrent" /></a><a href="https://github.com/blissdev"><img src="https://github.com/blissdev.png" width="60px" alt="Jordan Arentsen" /></a><a href="https://github.com/mislav"><img src="https://github.com/mislav.png" width="60px" alt="Mislav Marohnić" /></a><a href="https://github.com/aexvir"><img src="https://github.com/aexvir.png" width="60px" alt="Alex Viscreanu" /></a><a href="https://github.com/dbalatero"><img src="https://github.com/dbalatero.png" width="60px" alt="David Balatero" /></a><a href="https://github.com/moobar"><img src="https://github.com/moobar.png" width="60px" alt="" /></a><a href="https://github.com/majjoha"><img src="https://github.com/majjoha.png" width="60px" alt="Mathias Jean Johansen" /></a><a href="https://github.com/benelan"><img src="https://github.com/benelan.png" width="60px" alt="Ben Elan" /></a><a href="https://github.com/pawelduda"><img src="https://github.com/pawelduda.png" width="60px" alt="Paweł Duda" /></a><a href="https://github.com/slezica"><img src="https://github.com/slezica.png" width="60px" alt="Santiago Lezica" /></a><a href="https://github.com/pbwn"><img src="https://github.com/pbwn.png" width="60px" alt="" /></a><a href="https://github.com/timgluz"><img src="https://github.com/timgluz.png" width="60px" alt="Timo Sulg" /></a><a href="https://github.com/pyrho"><img src="https://github.com/pyrho.png" width="60px" alt="Damien Rajon" /></a><a href="https://github.com/ArtBIT"><img src="https://github.com/ArtBIT.png" width="60px" alt="ArtBIT" /></a><a href="https://github.com/da-moon"><img src="https://github.com/da-moon.png" width="60px" alt="" /></a><a href="https://github.com/jiangyinzuo"><img src="https://github.com/jiangyinzuo.png" width="60px" alt="Yinzuo Jiang" /></a><a href="https://github.com/hovissimo"><img src="https://github.com/hovissimo.png" width="60px" alt="Hovis" /></a><a href="https://github.com/dariusjonda"><img src="https://github.com/dariusjonda.png" width="60px" alt="Darius Jonda" /></a><a href="https://github.com/cristiand391"><img src="https://github.com/cristiand391.png" width="60px" alt="Cristian Dominguez" /></a><a href="https://github.com/eliangcs"><img src="https://github.com/eliangcs.png" width="60px" alt="Chang-Hung Liang" /></a><a href="https://github.com/asphaltbuffet"><img src="https://github.com/asphaltbuffet.png" width="60px" alt="Ben Lechlitner" /></a><a href="https://github.com/yash1th"><img src="https://github.com/yash1th.png" width="60px" alt="yash" /></a><a href="https://github.com/looshch"><img src="https://github.com/looshch.png" width="60px" alt="george looshch" /></a><a href="https://github.com/kg8m"><img src="https://github.com/kg8m.png" width="60px" alt="Takumi KAGIYAMA" /></a><a href="https://github.com/polm"><img src="https://github.com/polm.png" width="60px" alt="Paul OLeary McCann" /></a><a href="https://github.com/rbeeger"><img src="https://github.com/rbeeger.png" width="60px" alt="Robert Beeger" /></a><a href="https://github.com/veebch"><img src="https://github.com/veebch.png" width="60px" alt="VEEB Projects" /></a><a href="https://github.com/yowayb"><img src="https://github.com/yowayb.png" width="60px" alt="Yoway Buorn" /></a><a href="https://github.com/scalisi"><img src="https://github.com/scalisi.png" width="60px" alt="Josh Scalisi" /></a><a href="https://github.com/alecbcs"><img src="https://github.com/alecbcs.png" width="60px" alt="Alec Scott" /></a><a href="https://github.com/thnxdev"><img src="https://github.com/thnxdev.png" width="60px" alt="thanks.dev" /></a><a href="https://github.com/artursapek"><img src="https://github.com/artursapek.png" width="60px" alt="Artur Sapek" /></a><a href="https://github.com/ramnes"><img src="https://github.com/ramnes.png" width="60px" alt="Guillaume Gelin" /></a><a href="https://github.com/jyc"><img src="https://github.com/jyc.png" width="60px" alt="" /></a><a href="https://github.com/mrcnski"><img src="https://github.com/mrcnski.png" width="60px" alt="Marcin S." /></a><a href="https://github.com/roblevy"><img src="https://github.com/roblevy.png" width="60px" alt="Rob Levy" /></a><a href="https://github.com/glozow"><img src="https://github.com/glozow.png" width="60px" alt="Gloria Zhao" /></a><a href="https://github.com/wjt"><img src="https://github.com/wjt.png" width="60px" alt="Will Thompson" /></a><a href="https://github.com/toupeira"><img src="https://github.com/toupeira.png" width="60px" alt="Markus Koller" /></a><a href="https://github.com/rkpatel33"><img src="https://github.com/rkpatel33.png" width="60px" alt="" /></a><a href="https://github.com/jamesob"><img src="https://github.com/jamesob.png" width="60px" alt="jamesob" /></a><a href="https://github.com/jlebray"><img src="https://github.com/jlebray.png" width="60px" alt="Johan Le Bray" /></a><a href="https://github.com/panosl1"><img src="https://github.com/panosl1.png" width="60px" alt="Panos Lampropoulos" /></a><a href="https://github.com/bespinian"><img src="https://github.com/bespinian.png" width="60px" alt="bespinian" /></a><a href="https://github.com/scosu"><img src="https://github.com/scosu.png" width="60px" alt="Markus Schneider-Pargmann" /></a><a href="https://github.com/smithbm2316"><img src="https://github.com/smithbm2316.png" width="60px" alt="Ben Smith" /></a><a href="https://github.com/mywang-berk"><img src="https://github.com/mywang-berk.png" width="60px" alt="" /></a><a href="https://github.com/charlieegan3"><img src="https://github.com/charlieegan3.png" width="60px" alt="Charlie Egan" /></a><a href="https://github.com/thobbs"><img src="https://github.com/thobbs.png" width="60px" alt="Tyler Hobbs" /></a><a href="https://github.com/neilparikh"><img src="https://github.com/neilparikh.png" width="60px" alt="Neil Parikh" /></a><a href="https://github.com/gongahkia"><img src="https://github.com/gongahkia.png" width="60px" alt="Gabriel Ong" /></a><!-- sponsors -->

Table of Contents
-----------------

<!-- vim-markdown-toc GFM -->

* [Installation](#installation)
    * [Using Homebrew](#using-homebrew)
    * [Linux packages](#linux-packages)
    * [Windows packages](#windows-packages)
    * [Using git](#using-git)
    * [Binary releases](#binary-releases)
    * [Setting up shell integration](#setting-up-shell-integration)
    * [Vim/Neovim plugin](#vimneovim-plugin)
* [Upgrading fzf](#upgrading-fzf)
* [Building fzf](#building-fzf)
* [Usage](#usage)
    * [Using the finder](#using-the-finder)
    * [Display modes](#display-modes)
        * [`--height` mode](#--height-mode)
        * [`--tmux` mode](#--tmux-mode)
    * [Search syntax](#search-syntax)
    * [Environment variables](#environment-variables)
    * [Options](#options)
    * [Demo](#demo)
* [Examples](#examples)
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
    * [Previewing an image](#previewing-an-image)
* [Tips](#tips)
    * [Respecting `.gitignore`](#respecting-gitignore)
    * [Fish shell](#fish-shell)
    * [fzf Theme Playground](#fzf-theme-playground)
* [Related projects](#related-projects)
* [License](#license)

<!-- vim-markdown-toc -->

Installation
------------

### Using Homebrew

You can use [Homebrew](https://brew.sh/) (on macOS or Linux) to install fzf.

```sh
brew install fzf
```

> [!IMPORTANT]
> To set up shell integration (key bindings and fuzzy completion),
> see [the instructions below](#setting-up-shell-integration).

fzf is also available [via MacPorts][portfile]: `sudo port install fzf`

[portfile]: https://github.com/macports/macports-ports/blob/master/sysutils/fzf/Portfile

### Linux packages

| Package Manager | Linux Distribution      | Command                            |
| --------------- | ----------------------- | ---------------------------------- |
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
| Spack           |                         | `spack install fzf`                |
| XBPS            | Void Linux              | `sudo xbps-install -S fzf`         |
| Zypper          | openSUSE                | `sudo zypper install fzf`          |

> [!IMPORTANT]
> To set up shell integration (key bindings and fuzzy completion),
> see [the instructions below](#setting-up-shell-integration).

[![Packaging status](https://repology.org/badge/vertical-allrepos/fzf.svg?columns=3)](https://repology.org/project/fzf/versions)

### Windows packages

On Windows, fzf is available via [Chocolatey][choco], [Scoop][scoop],
[Winget][winget], and [MSYS2][msys2]:

| Package manager | Command                               |
| --------------- | ------------------------------------- |
| Chocolatey      | `choco install fzf`                   |
| Scoop           | `scoop install fzf`                   |
| Winget          | `winget install fzf`                  |
| MSYS2 (pacman)  | `pacman -S $MINGW_PACKAGE_PREFIX-fzf` |

[choco]: https://chocolatey.org/packages/fzf
[scoop]: https://github.com/ScoopInstaller/Main/blob/master/bucket/fzf.json
[winget]: https://github.com/microsoft/winget-pkgs/tree/master/manifests/j/junegunn/fzf
[msys2]: https://packages.msys2.org/base/mingw-w64-fzf

### Using git

Alternatively, you can "git clone" this repository to any directory and run
[install](https://github.com/junegunn/fzf/blob/master/install) script.

```sh
git clone --depth 1 https://github.com/junegunn/fzf.git ~/.fzf
~/.fzf/install
```

The install script will add lines to your shell configuration file to modify
`$PATH` and set up shell integration.

### Binary releases

You can download the official fzf binaries from the releases page.

* https://github.com/junegunn/fzf/releases

### Setting up shell integration

Add the following line to your shell configuration file.

* bash
  ```sh
  # Set up fzf key bindings and fuzzy completion
  eval "$(fzf --bash)"
  ```
* zsh
  ```sh
  # Set up fzf key bindings and fuzzy completion
  source <(fzf --zsh)
  ```
* fish
  ```fish
  # Set up fzf key bindings
  fzf --fish | source
  ```

> [!NOTE]
> `--bash`, `--zsh`, and `--fish` options are only available in fzf 0.48.0 or
> later. If you have an older version of fzf, or want finer control, you can
> source individual script files in the [/shell](/shell) directory. The
> location of the files may vary depending on the package manager you use.
> Please refer to the package documentation for more information.
> (e.g. `apt show fzf`)

> [!TIP]
> You can disable CTRL-T or ALT-C binding by setting `FZF_CTRL_T_COMMAND` or
> `FZF_ALT_C_COMMAND` to an empty string when sourcing the script.
> For example, to disable ALT-C binding:
>
> * bash: `FZF_ALT_C_COMMAND= eval "$(fzf --bash)"`
> * zsh: `FZF_ALT_C_COMMAND= source <(fzf --zsh)`
> * fish: `fzf --fish | FZF_ALT_C_COMMAND= source`
>
> Setting the variables after sourcing the script will have no effect.

### Vim/Neovim plugin

If you use [vim-plug](https://github.com/junegunn/vim-plug), add this to
your Vim configuration file:

```vim
Plug 'junegunn/fzf', { 'do': { -> fzf#install() } }
Plug 'junegunn/fzf.vim'
```

* `junegunn/fzf` provides the basic library functions
    * `fzf#install()` makes sure that you have the latest binary
* `junegunn/fzf.vim` is [a separate project](https://github.com/junegunn/fzf.vim)
  that provides a variety of useful commands

To learn more about the Vim integration, see [README-VIM.md](README-VIM.md).

> [!TIP]
> If you use Neovim and prefer Lua-based plugins, check out
> [fzf-lua](https://github.com/ibhagwan/fzf-lua).

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

Without STDIN pipe, fzf will traverse the file system under the current
directory to get the list of files.

```sh
vim $(fzf)
```

> [!NOTE]
> You can override the default behavior
> * Either by setting `$FZF_DEFAULT_COMMAND` to a command that generates the desired list
> * Or by setting `--walker`, `--walker-root`, and `--walker-skip` options in `$FZF_DEFAULT_OPTS`

> [!WARNING]
> A more robust solution would be to use `xargs` but we've presented
> the above as it's easier to grasp
> ```sh
> fzf --print0 | xargs -0 -o vim
> ```

> [!TIP]
> fzf also has the ability to turn itself into a different process.
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

### Display modes

fzf by default runs in fullscreen mode, but there are other display modes.

#### `--height` mode

With `--height HEIGHT[%]`, fzf will start below the cursor with the given height.

```sh
fzf --height 40%
```

`reverse` layout and `--border` goes well with this option.

```sh
fzf --height 40% --layout reverse --border
```

By prepending `~` to the height, you're setting the maximum height.

```sh
# Will take as few lines as possible to display the list
seq 3 | fzf --height ~100%
seq 3000 | fzf --height ~100%
```

Height value can be a negative number.

```sh
# Screen height - 3
fzf --height -3
```

#### `--tmux` mode

With `--tmux` option, fzf will start in a tmux popup.

```sh
# --tmux [center|top|bottom|left|right][,SIZE[%]][,SIZE[%]]

fzf --tmux center         # Center, 50% width and height
fzf --tmux 80%            # Center, 80% width and height
fzf --tmux 100%,50%       # Center, 100% width and 50% height
fzf --tmux left,40%       # Left, 40% width
fzf --tmux left,40%,90%   # Left, 40% width, 90% height
fzf --tmux top,40%        # Top, 40% height
fzf --tmux bottom,80%,40% # Bottom, 80% height, 40% height
```

`--tmux` is silently ignored when you're not on tmux.

> [!NOTE]
> If you're stuck with an old version of tmux that doesn't support popup,
> or if you want to open fzf in a regular tmux pane, check out
> [fzf-tmux](bin/fzf-tmux) script.

> [!TIP]
> You can add these options to `$FZF_DEFAULT_OPTS` so that they're applied by
> default. For example,
>
> ```sh
> # Open in tmux popup if on tmux, otherwise use --height mode
> export FZF_DEFAULT_OPTS='--height 40% --tmux bottom,40% --layout reverse --border top'
> ```

### Search syntax

Unless otherwise specified, fzf starts in "extended-search mode" where you can
type in multiple search terms delimited by spaces. e.g. `^music .mp3$ sbtrkt
!fire`

| Token     | Match type                              | Description                                  |
| --------- | --------------------------------------  | ------------------------------------------   |
| `sbtrkt`  | fuzzy-match                             | Items that match `sbtrkt`                    |
| `'wild`   | exact-match (quoted)                    | Items that include `wild`                    |
| `'wild'`  | exact-boundary-match (quoted both ends) | Items that include `wild` at word boundaries |
| `^music`  | prefix-exact-match                      | Items that start with `music`                |
| `.mp3$`   | suffix-exact-match                      | Items that end with `.mp3`                   |
| `!fire`   | inverse-exact-match                     | Items that do not include `fire`             |
| `!^music` | inverse-prefix-exact-match              | Items that do not start with `music`         |
| `!.mp3$`  | inverse-suffix-exact-match              | Items that do not end with `.mp3`            |

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
- `FZF_DEFAULT_OPTS`
    - Default options
    - e.g. `export FZF_DEFAULT_OPTS="--layout=reverse --inline-info"`
- `FZF_DEFAULT_OPTS_FILE`
    - If you prefer to manage default options in a file, set this variable to
      point to the location of the file
    - e.g. `export FZF_DEFAULT_OPTS_FILE=~/.fzfrc`

> [!WARNING]
> `FZF_DEFAULT_COMMAND` is not used by shell integration due to the
> slight difference in requirements.
>
> * `CTRL-T` runs `$FZF_CTRL_T_COMMAND` to get a list of files and directories
> * `ALT-C` runs `$FZF_ALT_C_COMMAND` to get a list of directories
> * `vim ~/**<tab>` runs `fzf_compgen_path()` with the prefix (`~/`) as the first argument
> * `cd foo**<tab>` runs `fzf_compgen_dir()` with the prefix (`foo`) as the first argument
>
> The available options are described later in this document.

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

Key bindings for command-line
-----------------------------

By [setting up shell integration](#setting-up-shell-integration), you can use
the following key bindings in bash, zsh, and fish.

- `CTRL-T` - Paste the selected files and directories onto the command-line
    - The list is generated using `--walker file,dir,follow,hidden` option
        - You can override the behavior by setting `FZF_CTRL_T_COMMAND` to a custom command that generates the desired list
        - Or you can set `--walker*` options in `FZF_CTRL_T_OPTS`
    - Set `FZF_CTRL_T_OPTS` to pass additional options to fzf
      ```sh
      # Preview file content using bat (https://github.com/sharkdp/bat)
      export FZF_CTRL_T_OPTS="
        --walker-skip .git,node_modules,target
        --preview 'bat -n --color=always {}'
        --bind 'ctrl-/:change-preview-window(down|hidden|)'"
      ```
    - Can be disabled by setting `FZF_CTRL_T_COMMAND` to an empty string when
      sourcing the script
- `CTRL-R` - Paste the selected command from history onto the command-line
    - If you want to see the commands in chronological order, press `CTRL-R`
      again which toggles sorting by relevance
    - Press `CTRL-/` or `ALT-/` to toggle line wrapping
    - Set `FZF_CTRL_R_OPTS` to pass additional options to fzf
      ```sh
      # CTRL-Y to copy the command into clipboard using pbcopy
      export FZF_CTRL_R_OPTS="
        --bind 'ctrl-y:execute-silent(echo -n {2..} | pbcopy)+abort'
        --color header:italic
        --header 'Press CTRL-Y to copy command into clipboard'"
      ```
- `ALT-C` - cd into the selected directory
    - The list is generated using `--walker dir,follow,hidden` option
    - Set `FZF_ALT_C_COMMAND` to override the default command
        - Or you can set `--walker-*` options in `FZF_ALT_C_OPTS`
    - Set `FZF_ALT_C_OPTS` to pass additional options to fzf
      ```sh
      # Print tree structure in the preview window
      export FZF_ALT_C_OPTS="
        --walker-skip .git,node_modules,target
        --preview 'tree -C {}'"
      ```
    - Can be disabled by setting `FZF_ALT_C_COMMAND` to an empty string when
      sourcing the script

Display modes for these bindings can be separately configured via
`FZF_{CTRL_T,CTRL_R,ALT_C}_OPTS` or globally via `FZF_DEFAULT_OPTS`.
(e.g. `FZF_CTRL_R_OPTS='--tmux bottom,60% --height 60% --border top'`)

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

# Use fd (https://github.com/sharkdp/fd) for listing path candidates.
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
[Highlight](https://gitlab.com/saalen/highlight):

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

> [!WARNING]
> Since fzf is a general-purpose text filter rather than a file finder, **it is
> not a good idea to add `--preview` option to your `$FZF_DEFAULT_OPTS`**.
>
> ```sh
> # *********************
> # ** DO NOT DO THIS! **
> # *********************
> export FZF_DEFAULT_OPTS='--preview "bat --style=numbers --color=always --line-range :500 {}"'
>
> # bat doesn't work with any input other than the list of files
> ps -ef | fzf
> seq 100 | fzf
> history | fzf
> ```

### Previewing an image

fzf can display images in the preview window using one of the following protocols:

* [Kitty graphics protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/)
* [iTerm2 inline images protocol](https://iterm2.com/documentation-images.html)
* [Sixel](https://en.wikipedia.org/wiki/Sixel)

See [bin/fzf-preview.sh](bin/fzf-preview.sh) script for more information.

```sh
fzf --preview 'fzf-preview.sh {}'
```

Tips
----

### Respecting `.gitignore`

You can use [fd](https://github.com/sharkdp/fd),
[ripgrep](https://github.com/BurntSushi/ripgrep), or [the silver
searcher](https://github.com/ggreer/the_silver_searcher) to traverse the file
system while respecting `.gitignore`.

```sh
# Feed the output of fd into fzf
fd --type f --strip-cwd-prefix | fzf

# Setting fd as the default source for fzf
export FZF_DEFAULT_COMMAND='fd --type f --strip-cwd-prefix'

# Now fzf (w/o pipe) will use the fd command to generate the list
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

### fzf Theme Playground

[fzf Theme Playground](https://vitormv.github.io/fzf-themes/) created by
[Vitor Mello](https://github.com/vitormv) is a webpage where you can
interactively create fzf themes.

Related projects
----------------

https://github.com/junegunn/fzf/wiki/Related-projects

[License](LICENSE)
------------------

The MIT License (MIT)

Copyright (c) 2013-2024 Junegunn Choi
