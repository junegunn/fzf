" Copyright (c) 2016 Junegunn Choi
"
" MIT License
"
" Permission is hereby granted, free of charge, to any person obtaining
" a copy of this software and associated documentation files (the
" "Software"), to deal in the Software without restriction, including
" without limitation the rights to use, copy, modify, merge, publish,
" distribute, sublicense, and/or sell copies of the Software, and to
" permit persons to whom the Software is furnished to do so, subject to
" the following conditions:
"
" The above copyright notice and this permission notice shall be
" included in all copies or substantial portions of the Software.
"
" THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
" EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
" MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
" NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
" LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
" OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
" WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

let s:default_height = '40%'
let s:fzf_go = expand('<sfile>:h:h').'/bin/fzf'
let s:install = expand('<sfile>:h:h').'/install'
let s:installed = 0
let s:fzf_tmux = expand('<sfile>:h:h').'/bin/fzf-tmux'

let s:cpo_save = &cpo
set cpo&vim

function! s:fzf_exec()
  if !exists('s:exec')
    if executable(s:fzf_go)
      let s:exec = s:fzf_go
    elseif executable('fzf')
      let s:exec = 'fzf'
    elseif !s:installed && executable(s:install) &&
          \ input('fzf executable not found. Download binary? (y/n) ') =~? '^y'
      redraw
      echo
      call s:warn('Downloading fzf binary. Please wait ...')
      let s:installed = 1
      call system(s:install.' --bin')
      return s:fzf_exec()
    else
      redraw
      throw 'fzf executable not found'
    endif
  endif
  return s:exec
endfunction

function! s:tmux_enabled()
  if has('gui_running')
    return 0
  endif

  if exists('s:tmux')
    return s:tmux
  endif

  let s:tmux = 0
  if exists('$TMUX') && executable(s:fzf_tmux)
    let output = system('tmux -V')
    let s:tmux = !v:shell_error && output >= 'tmux 1.7'
  endif
  return s:tmux
endfunction

function! s:shellesc(arg)
  return '"'.substitute(a:arg, '"', '\\"', 'g').'"'
endfunction

function! s:escape(path)
  return escape(a:path, ' $%#''"\')
endfunction

" Upgrade legacy options
function! s:upgrade(dict)
  let copy = copy(a:dict)
  if has_key(copy, 'tmux')
    let copy.down = remove(copy, 'tmux')
  endif
  if has_key(copy, 'tmux_height')
    let copy.down = remove(copy, 'tmux_height')
  endif
  if has_key(copy, 'tmux_width')
    let copy.right = remove(copy, 'tmux_width')
  endif
  return copy
endfunction

function! s:error(msg)
  echohl ErrorMsg
  echom a:msg
  echohl None
endfunction

function! s:warn(msg)
  echohl WarningMsg
  echom a:msg
  echohl None
endfunction

function! fzf#run(...) abort
try
  let oshell = &shell
  set shell=sh
  if has('nvim') && bufexists('term://*:FZF')
    call s:warn('FZF is already running!')
    return []
  endif
  let dict   = exists('a:1') ? s:upgrade(a:1) : {}
  let temps  = { 'result': tempname() }
  let optstr = get(dict, 'options', '')
  try
    let fzf_exec = s:fzf_exec()
  catch
    throw v:exception
  endtry

  if !has_key(dict, 'source') && !empty($FZF_DEFAULT_COMMAND)
    let dict.source = $FZF_DEFAULT_COMMAND
  endif

  if has_key(dict, 'source')
    let source = dict.source
    let type = type(source)
    if type == 1
      let prefix = source.'|'
    elseif type == 3
      let temps.input = tempname()
      call writefile(source, temps.input)
      let prefix = 'cat '.s:shellesc(temps.input).'|'
    else
      throw 'Invalid source type'
    endif
  else
    let prefix = ''
  endif
  let tmux = !has('nvim') && s:tmux_enabled() && s:splittable(dict)
  let command = prefix.(tmux ? s:fzf_tmux(dict) : fzf_exec).' '.optstr.' > '.temps.result

  if has('nvim')
    return s:execute_term(dict, command, temps)
  endif

  let ret = tmux ? s:execute_tmux(dict, command, temps) : s:execute(dict, command, temps)
  call s:popd(dict, ret)
  return ret
finally
  let &shell = oshell
endtry
endfunction

function! s:present(dict, ...)
  for key in a:000
    if !empty(get(a:dict, key, ''))
      return 1
    endif
  endfor
  return 0
endfunction

function! s:fzf_tmux(dict)
  let size = ''
  for o in ['up', 'down', 'left', 'right']
    if s:present(a:dict, o)
      let spec = a:dict[o]
      if (o == 'up' || o == 'down') && spec[0] == '~'
        let size = '-'.o[0].s:calc_size(&lines, spec[1:], a:dict)
      else
        " Legacy boolean option
        let size = '-'.o[0].(spec == 1 ? '' : spec)
      endif
      break
    endif
  endfor
  return printf('LINES=%d COLUMNS=%d %s %s %s --',
    \ &lines, &columns, s:fzf_tmux, size, (has_key(a:dict, 'source') ? '' : '-'))
endfunction

function! s:splittable(dict)
  return s:present(a:dict, 'up', 'down', 'left', 'right')
endfunction

function! s:pushd(dict)
  if s:present(a:dict, 'dir')
    let cwd = getcwd()
    if get(a:dict, 'prev_dir', '') ==# cwd
      return 1
    endif
    let a:dict.prev_dir = cwd
    execute 'lcd' s:escape(a:dict.dir)
    let a:dict.dir = getcwd()
    return 1
  endif
  return 0
endfunction

function! s:popd(dict, lines)
  " Since anything can be done in the sink function, there is no telling that
  " the change of the working directory was made by &autochdir setting.
  "
  " We use the following heuristic to determine whether to restore CWD:
  " - Always restore the current directory when &autochdir is disabled.
  "   FIXME This makes it impossible to change directory from inside the sink
  "   function when &autochdir is not used.
  " - In case of an error or an interrupt, a:lines will be empty.
  "   And it will be an array of a single empty string when fzf was finished
  "   without a match. In these cases, we presume that the change of the
  "   directory is not expected and should be undone.
  if has_key(a:dict, 'prev_dir') &&
        \ (!&autochdir || (empty(a:lines) || len(a:lines) == 1 && empty(a:lines[0])))
    execute 'lcd' s:escape(remove(a:dict, 'prev_dir'))
  endif
endfunction

function! s:xterm_launcher()
  let fmt = 'xterm -T "[fzf]" -bg "\%s" -fg "\%s" -geometry %dx%d+%d+%d -e bash -ic %%s'
  if has('gui_macvim')
    let fmt .= '&& osascript -e "tell application \"MacVim\" to activate"'
  endif
  return printf(fmt,
    \ synIDattr(hlID("Normal"), "bg"), synIDattr(hlID("Normal"), "fg"),
    \ &columns, &lines/2, getwinposx(), getwinposy())
endfunction
unlet! s:launcher
let s:launcher = function('s:xterm_launcher')

function! s:exit_handler(code, command, ...)
  if a:code == 130
    return 0
  elseif a:code > 1
    call s:error('Error running ' . a:command)
    if !empty(a:000)
      sleep
    endif
    return 0
  endif
  return 1
endfunction

function! s:execute(dict, command, temps) abort
  call s:pushd(a:dict)
  silent! !clear 2> /dev/null
  let escaped = escape(substitute(a:command, '\n', '\\n', 'g'), '%#')
  if has('gui_running')
    let Launcher = get(a:dict, 'launcher', get(g:, 'Fzf_launcher', get(g:, 'fzf_launcher', s:launcher)))
    let fmt = type(Launcher) == 2 ? call(Launcher, []) : Launcher
    let command = printf(fmt, "'".substitute(escaped, "'", "'\"'\"'", 'g')."'")
  else
    let command = escaped
  endif
  execute 'silent !'.command
  redraw!
  return s:exit_handler(v:shell_error, command) ? s:callback(a:dict, a:temps) : []
endfunction

function! s:execute_tmux(dict, command, temps) abort
  let command = a:command
  if s:pushd(a:dict)
    " -c '#{pane_current_path}' is only available on tmux 1.9 or above
    let command = 'cd '.s:escape(a:dict.dir).' && '.command
  endif

  call system(command)
  redraw!
  return s:exit_handler(v:shell_error, command) ? s:callback(a:dict, a:temps) : []
endfunction

function! s:calc_size(max, val, dict)
  if a:val =~ '%$'
    let size = a:max * str2nr(a:val[:-2]) / 100
  else
    let size = min([a:max, str2nr(a:val)])
  endif

  let srcsz = -1
  if type(get(a:dict, 'source', 0)) == type([])
    let srcsz = len(a:dict.source)
  endif

  let opts = get(a:dict, 'options', '').$FZF_DEFAULT_OPTS
  let margin = stridx(opts, '--inline-info') > stridx(opts, '--no-inline-info') ? 1 : 2
  return srcsz >= 0 ? min([srcsz + margin, size]) : size
endfunction

function! s:getpos()
  return {'tab': tabpagenr(), 'win': winnr(), 'cnt': winnr('$'), 'tcnt': tabpagenr('$')}
endfunction

function! s:split(dict)
  let directions = {
  \ 'up':    ['topleft', 'resize', &lines],
  \ 'down':  ['botright', 'resize', &lines],
  \ 'left':  ['vertical topleft', 'vertical resize', &columns],
  \ 'right': ['vertical botright', 'vertical resize', &columns] }
  let ppos = s:getpos()
  try
    for [dir, triple] in items(directions)
      let val = get(a:dict, dir, '')
      if !empty(val)
        let [cmd, resz, max] = triple
        if (dir == 'up' || dir == 'down') && val[0] == '~'
          let sz = s:calc_size(max, val[1:], a:dict)
        else
          let sz = s:calc_size(max, val, {})
        endif
        execute cmd sz.'new'
        execute resz sz
        return [ppos, {}]
      endif
    endfor
    if s:present(a:dict, 'window')
      execute a:dict.window
    else
      execute (tabpagenr()-1).'tabnew'
    endif
    return [ppos, { '&l:wfw': &l:wfw, '&l:wfh': &l:wfh }]
  finally
    setlocal winfixwidth winfixheight
  endtry
endfunction

function! s:execute_term(dict, command, temps) abort
  let [ppos, winopts] = s:split(a:dict)
  let fzf = { 'buf': bufnr('%'), 'ppos': ppos, 'dict': a:dict, 'temps': a:temps,
            \ 'name': 'FZF', 'winopts': winopts, 'command': a:command }
  function! fzf.switch_back(inplace)
    if a:inplace && bufnr('') == self.buf
      " FIXME: Can't re-enter normal mode from terminal mode
      " execute "normal! \<c-^>"
      b #
      " No other listed buffer
      if bufnr('') == self.buf
        enew
      endif
    endif
  endfunction
  function! fzf.on_exit(id, code)
    if s:getpos() == self.ppos " {'window': 'enew'}
      for [opt, val] in items(self.winopts)
        execute 'let' opt '=' val
      endfor
      call self.switch_back(1)
    else
      if bufnr('') == self.buf
        " We use close instead of bd! since Vim does not close the split when
        " there's no other listed buffer (nvim +'set nobuflisted')
        close
      endif
      execute 'tabnext' self.ppos.tab
      execute self.ppos.win.'wincmd w'
    endif

    if !s:exit_handler(a:code, self.command, 1)
      return
    endif

    call s:pushd(self.dict)
    let ret = []
    try
      let ret = s:callback(self.dict, self.temps)
      call self.switch_back(s:getpos() == self.ppos)
    finally
      call s:popd(self.dict, ret)
    endtry
  endfunction

  call s:pushd(a:dict)
  call termopen(a:command, fzf)
  call s:popd(a:dict, [])
  setlocal nospell bufhidden=wipe nobuflisted
  setf fzf
  startinsert
  return []
endfunction

function! s:callback(dict, temps) abort
let lines = []
try
  if filereadable(a:temps.result)
    let lines = readfile(a:temps.result)
    if has_key(a:dict, 'sink')
      for line in lines
        if type(a:dict.sink) == 2
          call a:dict.sink(line)
        else
          execute a:dict.sink s:escape(line)
        endif
      endfor
    endif
    if has_key(a:dict, 'sink*')
      call a:dict['sink*'](lines)
    endif
  endif

  for tf in values(a:temps)
    silent! call delete(tf)
  endfor
catch
  if stridx(v:exception, ':E325:') < 0
    echoerr v:exception
  endif
finally
  return lines
endtry
endfunction

let s:default_action = {
  \ 'ctrl-m': 'e',
  \ 'ctrl-t': 'tab split',
  \ 'ctrl-x': 'split',
  \ 'ctrl-v': 'vsplit' }

function! s:cmd_callback(lines) abort
  if empty(a:lines)
    return
  endif
  let key = remove(a:lines, 0)
  let cmd = get(s:action, key, 'e')
  if len(a:lines) > 1
    augroup fzf_swap
      autocmd SwapExists * let v:swapchoice='o'
            \| call s:warn('fzf: E325: swap file exists: '.expand('<afile>'))
    augroup END
  endif
  try
    let empty = empty(expand('%')) && line('$') == 1 && empty(getline(1)) && !&modified
    let autochdir = &autochdir
    set noautochdir
    for item in a:lines
      if empty
        execute 'e' s:escape(item)
        let empty = 0
      else
        execute cmd s:escape(item)
      endif
      if exists('#BufEnter') && isdirectory(item)
        doautocmd BufEnter
      endif
    endfor
  finally
    let &autochdir = autochdir
    silent! autocmd! fzf_swap
  endtry
endfunction

function! s:cmd(bang, ...) abort
  let s:action = get(g:, 'fzf_action', s:default_action)
  let args = extend(['--expect='.join(keys(s:action), ',')], a:000)
  let opts = {}
  if len(args) > 0 && isdirectory(expand(args[-1]))
    let opts.dir = substitute(remove(args, -1), '\\\(["'']\)', '\1', 'g')
  endif
  if !a:bang
    let opts.down = get(g:, 'fzf_height', get(g:, 'fzf_tmux_height', s:default_height))
  endif
  call fzf#run(extend({'options': join(args), 'sink*': function('<sid>cmd_callback')}, opts))
endfunction

command! -nargs=* -complete=dir -bang FZF call s:cmd(<bang>0, <f-args>)

let &cpo = s:cpo_save
unlet s:cpo_save

