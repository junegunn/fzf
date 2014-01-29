fzf - Fuzzy finder for your shell
=================================

fzf is a general-purpose fuzzy finder for your shell.

![](https://raw.github.com/junegunn/i/master/fzf.gif)

It was heavily inspired by [ctrlp.vim](https://github.com/kien/ctrlp.vim) and
the likes.

Requirements
------------

fzf requires Ruby (>= 1.8.5).

Installation
------------

Clone this repository and run
[install](https://github.com/junegunn/fzf/blob/master/install) script.

```sh
git clone https://github.com/junegunn/fzf.git ~/.fzf
~/.fzf/install
```

The script will setup:

- `fzf` executable
- Key bindings (`CTRL-T`, `CTRL-R`, etc.)
- Fuzzy auto-completion for bash

### Install as Vim plugin

You can use any Vim plugin manager to install fzf for Vim. If you don't use one,
I recommend you try [vim-plug](https://github.com/junegunn/vim-plug).

1. [Install vim-plug](https://github.com/junegunn/vim-plug#usage)
2. Edit your .vimrc

        call plug#begin()
        Plug 'junegunn/fzf'
        " ...
        call plug#end()

3. Run `:PlugInstall`

Usage
-----

```
usage: fzf [options]

  Options
    -m, --multi          Enable multi-select
    -x, --extended       Extended-search mode
    -q, --query=STR      Initial query
    -s, --sort=MAX       Maximum number of matched items to sort (default: 1000)
    +s, --no-sort        Do not sort the result. Keep the sequence unchanged.
    -i                   Case-insensitive match (default: smart-case match)
    +i                   Case-sensitive match
    +c, --no-color       Disable colors

  Environment variables
    FZF_DEFAULT_COMMAND  Default command to use when input is tty
    FZF_DEFAULT_OPTS     Defaults options. (e.g. "-x -m --sort 10000")
    FZF_MOUSE_ENABLED    Set to 0 to disable mouse
```

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

If you want to preserve the exact sequence of the input, provide `--no-sort` (or
`+s`) option.

```sh
history | fzf +s
```

### Key binding

Use CTRL-J and CTRL-K (or CTRL-N and CTRL-P) to change the selection, press
enter key to select the item. CTRL-C, CTRL-G, or ESC will terminate the finder.

The following readline key bindings should also work as expected.

- CTRL-A / CTRL-E
- CTRL-B / CTRL-F
- CTRL-W / CTRL-U
- ALT-B / ALT-F

If you enable multi-select mode with `-m` option, you can select multiple items
with TAB or Shift-TAB key.

You can also use mouse. Click on an item to select it or shift-click to select
multiple items. Use mouse wheel to move the cursor up and down.

### Extended-search mode

With `-x` or `--extended` option, fzf will start in "extended-search mode".

In this mode, you can specify multiple patterns delimited by spaces,
such as: `^music .mp3$ sbtrkt !rmx`

| Token    | Description                      | Match type           |
| -------- | -------------------------------- | -------------------- |
| `^music` | Items that start with `music`    | prefix-exact-match   |
| `.mp3$`  | Items that end with `.mp3`       | suffix-exact-match   |
| `sbtrkt` | Items that match `sbtrkt`        | fuzzy-match          |
| `!rmx`   | Items that do not match `rmx`    | inverse-fuzzy-match  |
| `'wild`  | Items that include `wild`        | exact-match (quoted) |
| `!'fire` | Items that do not include `fire` | inverse-exact-match  |

Useful examples
---------------

```sh
# vimf - Open selected file in Vim
vimf() {
  FILE=$(fzf) && vim "$FILE"
}

# fd - cd to selected directory
fd() {
  DIR=$(find ${1:-*} -path '*/\.*' -prune -o -type d -print 2> /dev/null | fzf) && cd "$DIR"
}

# fda - including hidden directories
fda() {
  DIR=$(find ${1:-.} -type d 2> /dev/null | fzf) && cd "$DIR"
}

# fh - repeat history
fh() {
  eval $(history | fzf +s | sed 's/ *[0-9]* *//')
}

# fkill - kill process
fkill() {
  ps -ef | sed 1d | fzf -m | awk '{print $2}' | xargs kill -${1:-9}
}
```

Key bindings for command line
-----------------------------

The install script will setup the following key bindings.

### bash

- `CTRL-T` - Paste the selected file path(s) into the command line
- `CTRL-R` - Paste the selected command from history into the command line

The source code can be found in `~/.fzf.bash`.

### zsh

- `CTRL-T` - Paste the selected file path(s) into the command line
- `CTRL-R` - Paste the selected command from history into the command line
- `ALT-C` - cd into the selected directory

The source code can be found in `~/.fzf.zsh`.

Auto-completion
---------------

Disclaimer: *Auto-completion feature is currently experimental, it can change
over time*

### bash

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

#### Settings

```sh
# Use ~~ as the trigger sequence instead of the default **
export FZF_COMPLETION_TRIGGER='~~'

# Options to fzf command
export FZF_COMPLETION_OPTS='+c -x'
```

### zsh

TODO :smiley:

(Pull requests are appreciated.)

Usage as Vim plugin
-------------------

If you install fzf as a Vim plugin, `:FZF` command will be added.

```vim
" Look for files under current directory
:FZF

" Look for files under your home directory
:FZF ~

" With options
:FZF --no-sort -m /tmp
```

You can override the source command which produces input to fzf.

```vim
let g:fzf_source = 'find . -type f'
```

And you can predefine default options to fzf command.

```vim
let g:fzf_options = '--no-color --extended'
```

For more advanced uses, you can call `fzf#run` function as follows.

```vim
:call fzf#run('tabedit', '-m +c')
```

Most of the time, you will prefer native Vim plugins with better integration
with Vim. The only reason one might consider using fzf in Vim is its speed. For
a very large list of files, fzf is significantly faster and it does not block.

Tips
----

### Faster startup with `--disable-gems` options

If you're running Ruby 1.9 or above, you can improve the startup time with
`--disable-gems` option to Ruby.

- `time ruby ~/bin/fzf -h`
    - 0.077 sec
- `time ruby --disable-gems ~/bin/fzf -h`
    - 0.025 sec

You can define fzf function with the option as follows:

```sh
fzf() {
  ruby --disable-gems ~/bin/fzf "$@"
}
export -f fzf
```

However, this is automatically set up in your .bashrc and .zshrc if you use the
bundled [install](https://github.com/junegunn/fzf/blob/master/install) script.

### Incorrect display on Ruby 1.8

It is reported that the output of fzf can become unreadable on some terminals
when it's running on Ruby 1.8. If you experience the problem, upgrade your Ruby
to 1.9 or above. Ruby 1.9 or above is also required for displaying Unicode
characters.

### Ranking algorithm

fzf sorts the result first by the length of the matched substring, then by the
length of the whole string. However it only does so when the number of matches
is less than the limit which is by default 1000, in order to avoid the cost of
sorting a large list and limit the response time of the query.

This limit can be adjusted with `-s` option, or with the environment variable
`FZF_DEFAULT_OPTS`.

```sh
export FZF_DEFAULT_OPTS="--sort 20000"
```

License
-------

MIT

Author
------

Junegunn Choi

