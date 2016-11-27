<img src="https://raw.githubusercontent.com/junegunn/i/master/fzf.png" height="170" alt="fzf - a command-line fuzzy finder"> [![travis-ci](https://travis-ci.org/junegunn/fzf.svg?branch=master)](https://travis-ci.org/junegunn/fzf)
===

fzf is a general-purpose command-line fuzzy finder.

![](https://raw.github.com/junegunn/i/master/fzf.gif)

Pros
----

- No dependencies
- Blazingly fast
- The most comprehensive feature set
- Flexible layout using tmux panes
- Batteries included
    - Vim/Neovim plugin, key bindings and fuzzy auto-completion

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

### Using git

Clone this repository and run
[install](https://github.com/junegunn/fzf/blob/master/install) script.

```sh
git clone --depth 1 https://github.com/junegunn/fzf.git ~/.fzf
~/.fzf/install
```

### Using Homebrew

On OS X, you can use [Homebrew](http://brew.sh/) to install fzf.

```sh
brew install fzf

# Install shell extensions
/usr/local/opt/fzf/install
```

### Vim plugin

You can manually add the directory to `&runtimepath` as follows,

```vim
" If installed using git
set rtp+=~/.fzf

" If installed using Homebrew
set rtp+=/usr/local/opt/fzf
```

But it's recommended that you use a plugin manager like
[vim-plug](https://github.com/junegunn/vim-plug).

```vim
Plug 'junegunn/fzf', { 'dir': '~/.fzf', 'do': './install --all' }
```

### Upgrading fzf

fzf is being actively developed and you might want to upgrade it once in a
while. Please follow the instruction below depending on the installation
method used.

- git: `cd ~/.fzf && git pull && ./install`
- brew: `brew update; brew reinstall fzf`
- vim-plug: `:PlugUpdate fzf`

### Windows

Pre-built binaries for Windows can be downloaded [here][bin]. However, other
components of the project may not work on Windows. You might want to consider
installing fzf on [Windows Subsystem for Linux][wsl] where everything runs
flawlessly.

[wsl]: https://blogs.msdn.microsoft.com/wsl/

Building fzf
------------

See [BUILD.md](BUILD.md).

Usage
-----

fzf will launch curses-based finder, read the list from STDIN, and write the
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

#### Search syntax

Unless otherwise specified, fzf starts in "extended-search mode" where you can
type in multiple search terms delimited by spaces. e.g. `^music .mp3$ sbtrkt
!fire`

| Token    | Match type                 | Description                       |
| -------- | -------------------------- | --------------------------------- |
| `sbtrkt` | fuzzy-match                | Items that match `sbtrkt`         |
| `^music` | prefix-exact-match         | Items that start with `music`     |
| `.mp3$`  | suffix-exact-match         | Items that end with `.mp3`        |
| `'wild`  | exact-match (quoted)       | Items that include `wild`         |
| `!fire`  | inverse-exact-match        | Items that do not include `fire`  |
| `!.mp3$` | inverse-suffix-exact-match | Items that do not end with `.mp3` |

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
    - e.g. `export FZF_DEFAULT_COMMAND='ag -g ""'`
- `FZF_DEFAULT_OPTS`
    - Default options
    - e.g. `export FZF_DEFAULT_OPTS="--reverse --inline-info"`

#### Options

See the man page (`man fzf`) for the full list of options.

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

Key bindings for command line
-----------------------------

The install script will setup the following key bindings for bash, zsh, and
fish.

- `CTRL-T` - Paste the selected files and directories onto the command line
    - Set `FZF_CTRL_T_COMMAND` to override the default command
    - Set `FZF_CTRL_T_OPTS` to pass additional options
- `CTRL-R` - Paste the selected command from history onto the command line
    - Sort is disabled by default to respect chronological ordering
    - Press `CTRL-R` again to toggle sort
    - Set `FZF_CTRL_R_OPTS` to pass additional options
- `ALT-C` - cd into the selected directory
    - Set `FZF_ALT_C_COMMAND` to override the default command
    - Set `FZF_ALT_C_OPTS` to pass additional options

If you're on a tmux session, fzf will start in a split pane. You may disable
this tmux integration by setting `FZF_TMUX` to 0, or change the height of the
pane with `FZF_TMUX_HEIGHT` (e.g. `20`, `50%`).

If you use vi mode on bash, you need to add `set -o vi` *before* `source
~/.fzf.bash` in your .bashrc, so that it correctly sets up key bindings for vi
mode.

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

Fuzzy completion for PIDs is provided for kill command. In this case
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

# Use ag instead of the default find command for listing candidates.
# - The first argument to the function is the base path to start traversal
# - Note that ag only lists files not directories
# - See the source code (completion.{bash,zsh}) for the details.
_fzf_compgen_path() {
  ag -g "" "$1"
}
```

#### Supported commands

On bash, fuzzy completion is enabled only for a predefined set of commands
(`complete | grep _fzf` to see the list). But you can enable it for other
commands as well like follows.

```sh
# There are also _fzf_path_completion and _fzf_dir_completion
complete -F _fzf_file_completion -o default -o bashdefault doge
```

Usage as Vim plugin
-------------------

This repository only enables basic integration with Vim. If you're looking for
more, check out [fzf.vim](https://github.com/junegunn/fzf.vim) project.

(Note: To use fzf in GVim, an external terminal emulator is required.)

#### `:FZF[!]`

If you have set up fzf for Vim, `:FZF` command will be added.

```vim
" Look for files under current directory
:FZF

" Look for files under your home directory
:FZF ~

" With options
:FZF --no-sort --reverse --inline-info /tmp

" Bang version starts in fullscreen instead of using tmux pane or Neovim split
:FZF!
```

Similarly to [ctrlp.vim](https://github.com/kien/ctrlp.vim), use enter key,
`CTRL-T`, `CTRL-X` or `CTRL-V` to open selected files in the current window,
in new tabs, in horizontal splits, or in vertical splits respectively.

Note that the environment variables `FZF_DEFAULT_COMMAND` and
`FZF_DEFAULT_OPTS` also apply here. Refer to [the wiki page][fzf-config] for
customization.

[fzf-config]: https://github.com/junegunn/fzf/wiki/Configuring-Vim-plugin

#### `fzf#run`

For more advanced uses, you can use `fzf#run([options])` function with the
following options.

| Option name                | Type          | Description                                                      |
| -------------------------- | ------------- | ---------------------------------------------------------------- |
| `source`                   | string        | External command to generate input to fzf (e.g. `find .`)        |
| `source`                   | list          | Vim list as input to fzf                                         |
| `sink`                     | string        | Vim command to handle the selected item (e.g. `e`, `tabe`)       |
| `sink`                     | funcref       | Reference to function to process each selected item              |
| `sink*`                    | funcref       | Similar to `sink`, but takes the list of output lines at once    |
| `options`                  | string        | Options to fzf                                                   |
| `dir`                      | string        | Working directory                                                |
| `up`/`down`/`left`/`right` | number/string | Use tmux pane with the given size (e.g. `20`, `50%`)             |
| `window` (*Neovim only*)   | string        | Command to open fzf window (e.g. `vertical aboveleft 30new`)     |
| `launcher`                 | string        | External terminal emulator to start fzf with (GVim only)         |
| `launcher`                 | funcref       | Function for generating `launcher` string (GVim only)            |

Examples can be found on [the wiki
page](https://github.com/junegunn/fzf/wiki/Examples-(vim)).

#### `fzf#wrap`

`fzf#wrap([name string,] [opts dict,] [fullscreen boolean])` is a helper
function that decorates the options dictionary so that it understands
`g:fzf_layout`, `g:fzf_action`, `g:fzf_colors`, and `g:fzf_history_dir` like
`:FZF`.

```vim
command! -bang MyStuff
  \ call fzf#run(fzf#wrap('my-stuff', {'dir': '~/my-stuff'}, <bang>0))
```

Tips
----

#### Rendering issues

If you have any rendering issues, check the following:

1. Make sure `$TERM` is correctly set. fzf will use 256-color only if it
   contains `256` (e.g. `xterm-256color`)
2. If you're on screen or tmux, `$TERM` should be either `screen` or
   `screen-256color`
3. Some terminal emulators (e.g. mintty) have problem displaying default
   background color and make some text unable to read. In that case, try
   `--black` option. And if it solves your problem, I recommend including it
   in `FZF_DEFAULT_OPTS` for further convenience.
4. If you still have problem, try `--no-256` option or even `--no-color`.

#### Respecting `.gitignore`, `.hgignore`, and `svn:ignore`

[ag](https://github.com/ggreer/the_silver_searcher) or
[pt](https://github.com/monochromegane/the_platinum_searcher) will do the
filtering:

```sh
# Feed the output of ag into fzf
ag -g "" | fzf

# Setting ag as the default source for fzf
export FZF_DEFAULT_COMMAND='ag -g ""'

# Now fzf (w/o pipe) will use ag instead of find
fzf

# To apply the command to CTRL-T as well
export FZF_CTRL_T_COMMAND="$FZF_DEFAULT_COMMAND"
```

If you don't want to exclude hidden files, use the following command:

```sh
export FZF_DEFAULT_COMMAND='ag --hidden --ignore .git -g ""'
```

#### `git ls-tree` for fast traversal

If you're running fzf in a large git repository, `git ls-tree` can boost up the
speed of the traversal.

```sh
export FZF_DEFAULT_COMMAND='
  (git ls-tree -r --name-only HEAD ||
   find . -path "*/\.*" -prune -o -type f -print -o -type l -print |
      sed s/^..//) 2> /dev/null'
```

#### Fish shell

It's [a known bug of fish](https://github.com/fish-shell/fish-shell/issues/1362)
that it doesn't allow reading from STDIN in command substitution, which means
simple `vim (fzf)` won't work as expected. The workaround is to store the result
of fzf to a temporary file.

```sh
fzf > $TMPDIR/fzf.result; and vim (cat $TMPDIR/fzf.result)
```

License
-------

[MIT](LICENSE)

Author
------

Junegunn Choi
