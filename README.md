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

Download [fzf executable](https://raw.github.com/junegunn/fzf/master/fzf) and
put it somewhere in your search $PATH.

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

### Install as Ruby gem

fzf can be installed as a Ruby gem

```
gem install fzf
```

It's a bit easier to install and update the script but the Ruby gem version
takes slightly longer to start.

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

  -s, --sort=MAX   Maximum number of matched items to sort. Default: 500
  +s, --no-sort    Keep the sequence unchanged.
  +i               Case-sensitive match
  +c, --no-color   Disable colors
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

Most of the time, you will prefer native Vim plugins with better integration
with Vim. The only reason one might consider using fzf in Vim is its speed. For
a very large list of files, fzf is significantly faster and it does not block.

Useful bash examples
--------------------

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
  DIR=$(find ${1:-*} -type d 2> /dev/null | fzf) && cd "$DIR"
}

# fh - repeat history
fh() {
  eval $(history | fzf +s | sed 's/ *[0-9]* *//')
}

# fkill - kill process
fkill() {
  ps -ef | sed 1d | fzf | awk '{print $2}' | xargs kill -${1:-9}
}

# (Assuming you don't use the default CTRL-T and CTRL-R)

# CTRL-T - Paste the selected file path into the command line
bind '"\er": redraw-current-line'
bind '"\C-t": " \C-u \C-a\C-k$(fzf)\e\C-e\C-y\C-a\C-y\ey\C-h\C-e\er"'

# CTRL-R - Paste the selected command from history into the command line
bind '"\C-r": " \C-e\C-u$(history | fzf +s | sed \"s/ *[0-9]* *//\")\e\C-e\er"'
```

License
-------

MIT

Author
------

Junegunn Choi

