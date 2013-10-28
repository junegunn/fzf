fzf - Fuzzy finder for your shell
=================================

fzf is a general-purpose fuzzy finder for your shell.

![](https://raw.github.com/junegunn/fzf/gif/fzf.gif)

It was heavily inspired by [ctrlp.vim](https://github.com/kien/ctrlp.vim) and
the likes.

Requirements
------------

fzf requires Ruby (>= 1.8.5).

Installation
------------

Download fzf executable and put it somewhere in your search $PATH.

```sh
mkdir -p ~/bin
wget https://raw.github.com/junegunn/fzf/master/fzf -O ~/bin/fzf
chmod +x ~/bin/fzf
```

Or you can just clone this repository and run
[install](https://github.com/junegunn/fzf/blob/master/install) script.

```sh
git clone https://github.com/junegunn/fzf.git
fzf/install
```

Make sure that ~/bin is included in $PATH.

```sh
export PATH=$PATH:~/bin
```

Install as Vim plugin
---------------------

You can use any plugin manager. If you don't use one, I recommend you try
[vim-plug](https://github.com/junegunn/vim-plug).

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

  -s, --sort=MAX   Maximum number of matched items to sort. Default: 500
  +s, --no-sort    Keep the sequence unchanged.
  +i               Case-sensitive match
```

fzf will launch curses-based finder, read the list from STDIN, and write the
selected item to STDOUT.

```sh
find * -type f | fzf > selected
```

Without STDIN pipe, fzf will use find command to fetch the list of
files (excluding hidden ones).

```sh
vim `fzf`
```

If you want to preserve the exact sequence of the input, provide `--no-sort` (or
`+s`) option.

```sh
history | fzf +s
```

### Key binding

Use CTRL-J and CTRL-K (or CTRL-N and CTRL-P) to change the selection, press
enter key to select the item. CTRL-C will terminate the finder.

The following readline key bindings should also work as expected.

- CTRL-A / CTRL-E
- CTRL-B / CTRL-F
- CTRL-W / CTRL-U

Usage as Vim plugin
-------------------

If you install fzf as a Vim plugin, `:FZF` command will be added.

```vim
:FZF
:FZF --no-sort
```

You can override the command which produces input to fzf.

```vim
let g:fzf_command = 'find . -type f'
```

Useful bash examples
--------------------

```sh
# vimf - Open selected file in Vim
vimf() {
  FILE=`fzf` && vim "$FILE"
}

# fd - cd to selected directory
fd() {
  DIR=`find ${1:-*} -path '*/\.*' -prune -o -type d -print 2> /dev/null | fzf` && cd "$DIR"
}

# fda - including hidden directories
fda() {
  DIR=`find ${1:-*} -type d 2> /dev/null | fzf` && cd "$DIR"
}

# fh - repeat history
fh() {
  eval $(history | fzf +s | sed 's/ *[0-9]* *//')
}

# fkill - kill process
fkill() {
  ps -ef | sed 1d | fzf | awk '{print $2}' | xargs kill -${1:-9}
}

# CTRL-T - Open fuzzy finder and paste the selected item to the command line
bind '"\er": redraw-current-line'
bind '"\C-t": " \C-u \C-a\C-k$(fzf)\e\C-e\C-y\C-a\C-y\ey\C-h\C-e\er"'
```

License
-------

MIT

Author
------

Junegunn Choi

