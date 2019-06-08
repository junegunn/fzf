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
let g:fzf_layout = { 'window': '10new' }

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

For more advanced uses, you can use `fzf#run([options])` function.

`fzf#run()` function is the core of Vim integration. It takes a single
dictionary argument. At the very least, specify `sink` option to tell what it
should do with the selected entry.

```vim
call fzf#run({'sink': 'e'})
```

Without `source`, fzf will use find command (or `$FZF_DEFAULT_COMMAND` if
defined) to list the files under the current directory. When you select one,
it will open it with `:e` command. If you want to open it in a new tab, you
can pass `:tabedit` command instead as the sink.

```vim
call fzf#run({'sink': 'tabedit'})
```

fzf allows you to select multiple entries with `--multi` (or `-m`) option, and
you can change its bottom-up layout with `--reverse` option. Such options can
be specified as `options`.

```vim
call fzf#run({'sink': 'tabedit', 'options': '--multi --reverse'})
```

Instead of using the default find command, you can use any shell command as
the source. This will list the files managed by git.

```vim
call fzf#run({'source': 'git ls-files', 'sink': 'e'})
```

Pass a layout option if you don't want fzf window to take up the entire screen.

```vim
" up / down / left / right / window are allowed
call fzf#run({'source': 'git ls-files', 'sink': 'e', 'right': '40%'})
call fzf#run({'source': 'git ls-files', 'sink': 'e', 'window': '30vnew'})
```

`source` doesn't have to be an external shell command, you can pass a Vim
array as the source. In the following example, we use the names of the open
buffers as the source.

```vim
call fzf#run({'source': map(filter(range(1, bufnr('$')), 'buflisted(v:val)'),
            \               'bufname(v:val)'),
            \ 'sink': 'e', 'down': '30%'})
```

Or the names of color schemes.

```vim
call fzf#run({'source': map(split(globpath(&rtp, 'colors/*.vim')),
            \               'fnamemodify(v:val, ":t:r")'),
            \ 'sink': 'colo', 'left': '25%'})
```

The following table shows the available options.

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

`:FZF` command provided by default knows how to handle `CTRL-T`, `CTRL-X`, and
`CTRL-V` and opens the selected file in a new tab, in a horizontal split, or
in a vertical split respectively. And these key bindings can be configured via
`g:fzf_action`. This is implemented using `--expect` option of fzf and the
smart sink function. It also understands `g:fzf_colors`, `g:fzf_layout` and
`g:fzf_history_dir`. However, `fzf#run` doesn't know about any of these
options.

By *"wrapping"* your options dictionary with `fzf#wrap` before passing it to
`fzf#run`, you can make your command also support the options.

```vim
" Usage:
"   fzf#wrap([name string,] [opts dict,] [fullscreen boolean])

" This command now supports CTRL-T, CTRL-V, and CTRL-X key bindings
" and opens fzf according to g:fzf_layout setting.
command! Buffers call fzf#run(fzf#wrap(
    \ {'source': map(range(1, bufnr('$')), 'bufname(v:val)')}))

" This extends the above example to open fzf in fullscreen
" when the command is run with ! suffix (Buffers!)
command! -bang Buffers call fzf#run(fzf#wrap(
    \ {'source': map(range(1, bufnr('$')), 'bufname(v:val)')}, <bang>0))

" You can optionally pass the name of the command as the first argument to
" fzf#wrap to make it work with g:fzf_history_dir
command! -bang Buffers call fzf#run(fzf#wrap('buffers',
    \ {'source': map(range(1, bufnr('$')), 'bufname(v:val)')}, <bang>0))
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

[License](LICENSE)
------------------

The MIT License (MIT)

Copyright (c) 2017 Junegunn Choi
