FZF Vim integration
===================

This repository only enables basic integration with Vim. If you're looking for
more, check out [fzf.vim](https://github.com/junegunn/fzf.vim) project.

(Note: To use fzf in GVim, an external terminal emulator is required.)

`:FZF[!]`
---------

If you have set up fzf for Vim, `:FZF` command will be added.

```vim
" Look for files under current directory
:FZF

" Look for files under your home directory
:FZF ~

" With options
:FZF --no-sort --reverse --inline-info /tmp

" Bang version starts fzf in fullscreen mode
:FZF!
```

Similarly to [ctrlp.vim](https://github.com/kien/ctrlp.vim), use enter key,
`CTRL-T`, `CTRL-X` or `CTRL-V` to open selected files in the current window,
in new tabs, in horizontal splits, or in vertical splits respectively.

Note that the environment variables `FZF_DEFAULT_COMMAND` and
`FZF_DEFAULT_OPTS` also apply here.

### Configuration

- `g:fzf_action`
    - Customizable extra key bindings for opening selected files in different ways
- `g:fzf_layout`
    - Determines the size and position of fzf window
- `g:fzf_colors`
    - Customizes fzf colors to match the current color scheme
- `g:fzf_history_dir`
    - Enables history feature
- `g:fzf_launcher`
    - (Only in GVim) Terminal emulator to open fzf with
    - `g:Fzf_launcher` for function reference

#### Examples

```vim
" This is the default extra key bindings
let g:fzf_action = {
  \ 'ctrl-t': 'tab split',
  \ 'ctrl-x': 'split',
  \ 'ctrl-v': 'vsplit' }

" An action can be a reference to a function that processes selected lines
function! s:build_quickfix_list(lines)
  call setqflist(map(copy(a:lines), '{ "filename": v:val }'))
  copen
  cc
endfunction

let g:fzf_action = {
  \ 'ctrl-q': function('s:build_quickfix_list'),
  \ 'ctrl-t': 'tab split',
  \ 'ctrl-x': 'split',
  \ 'ctrl-v': 'vsplit' }

" Default fzf layout
" - down / up / left / right
let g:fzf_layout = { 'down': '~40%' }

" You can set up fzf window using a Vim command (Neovim or latest Vim 8 required)
let g:fzf_layout = { 'window': 'enew' }
let g:fzf_layout = { 'window': '-tabnew' }
let g:fzf_layout = { 'window': '10split enew' }

" Customize fzf colors to match your color scheme
let g:fzf_colors =
\ { 'fg':      ['fg', 'Normal'],
  \ 'bg':      ['bg', 'Normal'],
  \ 'hl':      ['fg', 'Comment'],
  \ 'fg+':     ['fg', 'CursorLine', 'CursorColumn', 'Normal'],
  \ 'bg+':     ['bg', 'CursorLine', 'CursorColumn'],
  \ 'hl+':     ['fg', 'Statement'],
  \ 'info':    ['fg', 'PreProc'],
  \ 'border':  ['fg', 'Ignore'],
  \ 'prompt':  ['fg', 'Conditional'],
  \ 'pointer': ['fg', 'Exception'],
  \ 'marker':  ['fg', 'Keyword'],
  \ 'spinner': ['fg', 'Label'],
  \ 'header':  ['fg', 'Comment'] }

" Enable per-command history.
" CTRL-N and CTRL-P will be automatically bound to next-history and
" previous-history instead of down and up. If you don't like the change,
" explicitly bind the keys to down and up in your $FZF_DEFAULT_OPTS.
let g:fzf_history_dir = '~/.local/share/fzf-history'
```

`fzf#run`
---------

For more advanced uses, you can use `fzf#run([options])` function with the
following options.

| Option name                | Type          | Description                                                      |
| -------------------------- | ------------- | ---------------------------------------------------------------- |
| `source`                   | string        | External command to generate input to fzf (e.g. `find .`)        |
| `source`                   | list          | Vim list as input to fzf                                         |
| `sink`                     | string        | Vim command to handle the selected item (e.g. `e`, `tabe`)       |
| `sink`                     | funcref       | Reference to function to process each selected item              |
| `sink*`                    | funcref       | Similar to `sink`, but takes the list of output lines at once    |
| `options`                  | string/list   | Options to fzf                                                   |
| `dir`                      | string        | Working directory                                                |
| `up`/`down`/`left`/`right` | number/string | Use tmux pane with the given size (e.g. `20`, `50%`)             |
| `window` (Vim 8 / Neovim)  | string        | Command to open fzf window (e.g. `vertical aboveleft 30new`)     |
| `launcher`                 | string        | External terminal emulator to start fzf with (GVim only)         |
| `launcher`                 | funcref       | Function for generating `launcher` string (GVim only)            |

`options` entry can be either a string or a list. For simple cases, string
should suffice, but prefer to use list type if you're concerned about escaping
issues on different platforms.

```vim
call fzf#run({'options': '--reverse --prompt "C:\\Program Files\\"'})
call fzf#run({'options': ['--reverse', '--prompt', 'C:\Program Files\']})
```

`fzf#wrap`
----------

`fzf#wrap([name string,] [opts dict,] [fullscreen boolean])` is a helper
function that decorates the options dictionary so that it understands
`g:fzf_layout`, `g:fzf_action`, `g:fzf_colors`, and `g:fzf_history_dir` like
`:FZF`.

```vim
command! -bang MyStuff
  \ call fzf#run(fzf#wrap('my-stuff', {'dir': '~/my-stuff'}, <bang>0))
```

fzf inside terminal buffer
--------------------------

The latest versions of Vim and Neovim include builtin terminal emulator
(`:terminal`) and fzf will start in a terminal buffer in the following cases:

- On Neovim
- On GVim
- On Terminal Vim with the non-default layout
    - `call fzf#run({'left': '30%'})` or `let g:fzf_layout = {'left': '30%'}`

### Hide statusline

When fzf starts in a terminal buffer, you may want to hide the statusline of
the containing buffer.

```vim
autocmd! FileType fzf
autocmd  FileType fzf set laststatus=0 noshowmode noruler
  \| autocmd BufLeave <buffer> set laststatus=2 showmode ruler
```

GVim
----

With the latest version of GVim, fzf will start inside the builtin terminal
emulator of Vim. Please note that this terminal feature of Vim is still young
and unstable and you may run into some issues.

If you have an older version of GVim, you need an external terminal emulator
to start fzf with. `xterm` command is used by default, but you can customize
it with `g:fzf_launcher`.

```vim
" This is the default. %s is replaced with fzf command
let g:fzf_launcher = 'xterm -e bash -ic %s'

" Use urxvt instead
let g:fzf_launcher = 'urxvt -geometry 120x30 -e sh -c %s'
```

If you're running MacVim on OSX, I recommend you to use iTerm2 as the
launcher. Refer to the [this wiki page][macvim-iterm2] to see how to set up.

[macvim-iterm2]: https://github.com/junegunn/fzf/wiki/On-MacVim-with-iTerm2

[License](LICENSE)
------------------

The MIT License (MIT)

Copyright (c) 2017 Junegunn Choi
