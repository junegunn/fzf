<img src="https://raw.githubusercontent.com/junegunn/i/master/fzf.png" height="170" alt="fzf - a command-line fuzzy finder"> [![travis-ci](https://travis-ci.org/junegunn/fzf.svg?branch=master)](https://travis-ci.org/junegunn/fzf) <a href="http://flattr.com/thing/3115381/junegunnfzf-on-GitHub" target="_blank"><img src="http://api.flattr.com/button/flattr-badge-large.png" alt="Flattr this" title="Flattr this" border="0" /></a>
===

fzf is a general-purpose command-line fuzzy finder.

![](https://raw.github.com/junegunn/i/master/fzf.gif)

Pros
----

- No dependencies
- Blazingly fast
    - e.g. `locate / | fzf`
- Flexible layout
    - Runs in fullscreen or in horizontal/vertical split using tmux
- The most comprehensive feature set
    - Try `fzf --help` and be surprised
- Batteries included
    - Vim/Neovim plugin, key bindings and fuzzy auto-completion

Installation
------------

fzf project consists of the followings:

- `fzf` executable
- `fzf-tmux` script for launching fzf in a tmux pane
- Shell extensions
    - Key bindings (`CTRL-T`, `CTRL-R`, and `ALT-C`) (bash, zsh, fish)
    - Fuzzy auto-completion (bash, zsh)
- Vim/Neovim plugin

You can [download fzf executable][bin] alone, but it's recommended that you
install the extra stuff using the attached install script.

[bin]: https://github.com/junegunn/fzf-bin/releases

#### Using git (recommended)

Clone this repository and run
[install](https://github.com/junegunn/fzf/blob/master/install) script.

```sh
git clone --depth 1 https://github.com/junegunn/fzf.git ~/.fzf
~/.fzf/install
```

#### Using Homebrew

On OS X, you can use [Homebrew](http://brew.sh/) to install fzf.

```sh
brew reinstall --HEAD fzf

# Install shell extensions
/usr/local/Cellar/fzf/HEAD/install
```

#### Install as Vim plugin

Once you have cloned the repository, add the following line to your .vimrc.

```vim
set rtp+=~/.fzf
```

Or you can have [vim-plug](https://github.com/junegunn/vim-plug) manage fzf
(recommended):

```vim
Plug 'junegunn/fzf', { 'dir': '~/.fzf', 'do': 'yes \| ./install' }
```

#### Upgrading fzf

fzf is being actively developed and you might want to upgrade it once in a
while. Please follow the instruction below depending on the installation
method.

- git: `cd ~/.fzf && git pull && ./install`
- brew: `brew reinstall --HEAD fzf`
- vim-plug: `:PlugUpdate fzf`

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

- `CTRL-J` / `CTRL-K` (or `CTRL-N` / `CTRL-P)` to move cursor up and down
- `Enter` key to select the item, `CTRL-C` / `CTRL-G` / `ESC` to exit
- On multi-select mode (`-m`), `TAB` and `Shift-TAB` to mark multiple items
- Emacs style key bindings
- Mouse: scroll, click, double-click; shift-click and shift-scroll on
  multi-select mode

#### Extended-search mode

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

If you don't need fuzzy matching and do not wish to "quote" every word, start
fzf with `-e` or `--extended-exact` option.

Examples
--------

Many useful examples can be found on [the wiki
page](https://github.com/junegunn/fzf/wiki/examples). Feel free to add your
own as well.

Key bindings for command line
-----------------------------

The install script will setup the following key bindings for bash, zsh, and
fish.

- `CTRL-T` - Paste the selected file path(s) into the command line
- `CTRL-R` - Paste the selected command from history into the command line
    - Sort is disabled by default to respect chronological ordering
    - Press `CTRL-R` again to toggle sort
- `ALT-C` - cd into the selected directory

If you're on a tmux session, fzf will start in a split pane. You may disable
this tmux integration by setting `FZF_TMUX` to 0, or change the height of the
pane with `FZF_TMUX_HEIGHT` (e.g. `20`, `50%`).

If you use vi mode on bash, you need to add `set -o vi` *before* `source
~/.fzf.bash` in your .bashrc, so that it correctly sets up key bindings for vi
mode.

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
```

Usage as Vim plugin
-------------------

(Note: To use fzf in GVim, an external terminal emulator is required.)

#### `:FZF[!]`

If you have set up fzf for Vim, `:FZF` command will be added.

```vim
" Look for files under current directory
:FZF

" Look for files under your home directory
:FZF ~

" With options
:FZF --no-sort -m /tmp

" Bang version starts in fullscreen instead of using tmux pane or Neovim split
:FZF!
```

Similarly to [ctrlp.vim](https://github.com/kien/ctrlp.vim), use enter key,
`CTRL-T`, `CTRL-X` or `CTRL-V` to open selected files in the current window,
in new tabs, in horizontal splits, or in vertical splits respectively.

Note that the environment variables `FZF_DEFAULT_COMMAND` and
`FZF_DEFAULT_OPTS` also apply here. Refer to [the wiki page][fzf-config] for
customization.

[fzf-config]: https://github.com/junegunn/fzf/wiki/Configuring-FZF-command-(vim)

#### `fzf#run([options])`

For more advanced uses, you can call `fzf#run()` function which returns the list
of the selected items.

`fzf#run()` may take an options-dictionary:

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

_However on Neovim `fzf#run` is asynchronous and does not return values so you
should use `sink` or `sink*` to process the output from fzf._

##### Examples

If `sink` option is not given, `fzf#run` will simply return the list.

```vim
let items = fzf#run({ 'options': '-m +c', 'dir': '~', 'source': 'ls' })
```

But if `sink` is given as a string, the command will be executed for each
selected item.

```vim
" Each selected item will be opened in a new tab
let items = fzf#run({ 'sink': 'tabe', 'options': '-m +c', 'dir': '~', 'source': 'ls' })
```

We can also use a Vim list as the source as follows:

```vim
" Choose a color scheme with fzf
nnoremap <silent> <Leader>C :call fzf#run({
\   'source':
\     map(split(globpath(&rtp, "colors/*.vim"), "\n"),
\         "substitute(fnamemodify(v:val, ':t'), '\\..\\{-}$', '', '')"),
\   'sink':     'colo',
\   'options':  '+m',
\   'left':     20,
\   'launcher': 'xterm -geometry 20x30 -e bash -ic %s'
\ })<CR>
```

`sink` option can be a function reference. The following example creates a
handy mapping that selects an open buffer.

```vim
" List of buffers
function! s:buflist()
  redir => ls
  silent ls
  redir END
  return split(ls, '\n')
endfunction

function! s:bufopen(e)
  execute 'buffer' matchstr(a:e, '^[ 0-9]*')
endfunction

nnoremap <silent> <Leader><Enter> :call fzf#run({
\   'source':  reverse(<sid>buflist()),
\   'sink':    function('<sid>bufopen'),
\   'options': '+m',
\   'down':    len(<sid>buflist()) + 2
\ })<CR>
```

More examples can be found on [the wiki
page](https://github.com/junegunn/fzf/wiki/Examples-(vim)).

Tips
----

#### Rendering issues

If you have any rendering issues, check the followings:

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
ag -l -g "" | fzf

# Setting ag as the default source for fzf
export FZF_DEFAULT_COMMAND='ag -l -g ""'

# Now fzf (w/o pipe) will use ag instead of find
fzf
```

#### `git ls-tree` for fast traversal

If you're running fzf in a large git repository, `git ls-tree` can boost up the
speed of the traversal.

```sh
export FZF_DEFAULT_COMMAND='
  (git ls-tree -r --name-only HEAD ||
   find * -name ".*" -prune -o -type f -print -o -type l -print) 2> /dev/null'
```

#### Fish shell

It's [a known bug of fish](https://github.com/fish-shell/fish-shell/issues/1362)
that it doesn't allow reading from STDIN in command substitution, which means
simple `vim (fzf)` won't work as expected. The workaround is to store the result
of fzf to a temporary file.

```sh
fzf > $TMPDIR/fzf.result; and vim (cat $TMPDIR/fzf.result)
```

#### Handling UTF-8 NFD paths on OSX

Use iconv to convert NFD paths to NFC:

```sh
find . | iconv -f utf-8-mac -t utf8//ignore | fzf
```

License
-------

[MIT](LICENSE)

Author
------

Junegunn Choi
