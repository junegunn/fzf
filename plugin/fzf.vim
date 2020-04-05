" Copyright (c) 2017 Junegunn Choi
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

if exists('g:loaded_fzf')
  finish
endif
let g:loaded_fzf = 1

let s:is_win = has('win32') || has('win64')
if s:is_win && &shellslash
  set noshellslash
  let s:base_dir = expand('<sfile>:h:h')
  set shellslash
else
  let s:base_dir = expand('<sfile>:h:h')
endif
if s:is_win
  let s:term_marker = '&::FZF'

  function! s:fzf_call(fn, ...)
    let shellslash = &shellslash
    try
      set noshellslash
      return call(a:fn, a:000)
    finally
      let &shellslash = shellslash
    endtry
  endfunction

  " Use utf-8 for fzf.vim commands
  " Return array of shell commands for cmd.exe
  function! s:enc_to_cp(str)
    if !has('iconv')
      return a:str
    endif
    if !exists('s:codepage')
      let s:codepage = libcallnr('kernel32.dll', 'GetACP', 0)
    endif
    return iconv(a:str, &encoding, 'cp'.s:codepage)
  endfunction
  function! s:wrap_cmds(cmds)
    return map([
      \ '@echo off',
      \ 'setlocal enabledelayedexpansion']
    \ + (has('gui_running') ? ['set TERM= > nul'] : [])
    \ + (type(a:cmds) == type([]) ? a:cmds : [a:cmds])
    \ + ['endlocal'],
    \ '<SID>enc_to_cp(v:val."\r")')
  endfunction
else
  let s:term_marker = ";#FZF"

  function! s:fzf_call(fn, ...)
    return call(a:fn, a:000)
  endfunction

  function! s:wrap_cmds(cmds)
    return a:cmds
  endfunction

  function! s:enc_to_cp(str)
    return a:str
  endfunction
endif

function! s:shellesc_cmd(arg)
  let escaped = substitute(a:arg, '[&|<>()@^]', '^&', 'g')
  let escaped = substitute(escaped, '%', '%%', 'g')
  let escaped = substitute(escaped, '"', '\\^&', 'g')
  let escaped = substitute(escaped, '\(\\\+\)\(\\^\)', '\1\1\2', 'g')
  return '^"'.substitute(escaped, '\(\\\+\)$', '\1\1', '').'^"'
endfunction

function! fzf#shellescape(arg, ...)
  let shell = get(a:000, 0, s:is_win ? 'cmd.exe' : 'sh')
  if shell =~# 'cmd.exe$'
    return s:shellesc_cmd(a:arg)
  endif
  return s:fzf_call('shellescape', a:arg)
endfunction

function! s:fzf_getcwd()
  return s:fzf_call('getcwd')
endfunction

function! s:fzf_fnamemodify(fname, mods)
  return s:fzf_call('fnamemodify', a:fname, a:mods)
endfunction

function! s:fzf_expand(fmt)
  return s:fzf_call('expand', a:fmt, 1)
endfunction

function! s:fzf_tempname()
  return s:fzf_call('tempname')
endfunction

let s:default_layout = { 'down': '~40%' }
let s:layout_keys = ['window', 'tmux', 'up', 'down', 'left', 'right']
let s:fzf_go = s:base_dir.'/bin/fzf'
let s:fzf_tmux = s:base_dir.'/bin/fzf-tmux'

let s:cpo_save = &cpo
set cpo&vim

function! fzf#install()
  if s:is_win && !has('win32unix')
    let script = s:base_dir.'/install.ps1'
    if !filereadable(script)
      throw script.' not found'
    endif
    let script = 'powershell -ExecutionPolicy Bypass -file ' . script
  else
    let script = s:base_dir.'/install'
    if !executable(script)
      throw script.' not found'
    endif
    let script .= ' --bin'
  endif

  call s:warn('Running fzf installer ...')
  call system(script)
  if v:shell_error
    throw 'Failed to download fzf: '.script
  endif
endfunction

function! s:fzf_exec()
  if !exists('s:exec')
    if executable(s:fzf_go)
      let s:exec = s:fzf_go
    elseif executable('fzf')
      let s:exec = 'fzf'
    elseif input('fzf executable not found. Download binary? (y/n) ') =~? '^y'
      redraw
      call fzf#install()
      return s:fzf_exec()
    else
      redraw
      throw 'fzf executable not found'
    endif
  endif
  return fzf#shellescape(s:exec)
endfunction

function! s:tmux_enabled()
  if has('gui_running') || !exists('$TMUX')
    return 0
  endif

  if exists('s:tmux')
    return s:tmux
  endif

  let s:tmux = 0
  if !executable(s:fzf_tmux)
    if executable('fzf-tmux')
      let s:fzf_tmux = 'fzf-tmux'
    else
      return 0
    endif
  endif

  let output = system('tmux -V')
  let s:tmux = !v:shell_error && output >= 'tmux 1.7'
  return s:tmux
endfunction

function! s:escape(path)
  let path = fnameescape(a:path)
  return s:is_win ? escape(path, '$') : path
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

function! s:has_any(dict, keys)
  for key in a:keys
    if has_key(a:dict, key)
      return 1
    endif
  endfor
  return 0
endfunction

function! s:open(cmd, target)
  if stridx('edit', a:cmd) == 0 && s:fzf_fnamemodify(a:target, ':p') ==# s:fzf_expand('%:p')
    return
  endif
  execute a:cmd s:escape(a:target)
endfunction

function! s:common_sink(action, lines) abort
  if len(a:lines) < 2
    return
  endif
  let key = remove(a:lines, 0)
  let Cmd = get(a:action, key, 'e')
  if type(Cmd) == type(function('call'))
    return Cmd(a:lines)
  endif
  if len(a:lines) > 1
    augroup fzf_swap
      autocmd SwapExists * let v:swapchoice='o'
            \| call s:warn('fzf: E325: swap file exists: '.s:fzf_expand('<afile>'))
    augroup END
  endif
  try
    let empty = empty(s:fzf_expand('%')) && line('$') == 1 && empty(getline(1)) && !&modified
    let autochdir = &autochdir
    set noautochdir
    for item in a:lines
      if empty
        execute 'e' s:escape(item)
        let empty = 0
      else
        call s:open(Cmd, item)
      endif
      if !has('patch-8.0.0177') && !has('nvim-0.2') && exists('#BufEnter')
            \ && isdirectory(item)
        doautocmd BufEnter
      endif
    endfor
  catch /^Vim:Interrupt$/
  finally
    let &autochdir = autochdir
    silent! autocmd! fzf_swap
  endtry
endfunction

function! s:get_color(attr, ...)
  let gui = !s:is_win && !has('win32unix') && has('termguicolors') && &termguicolors
  let fam = gui ? 'gui' : 'cterm'
  let pat = gui ? '^#[a-f0-9]\+' : '^[0-9]\+$'
  for group in a:000
    let code = synIDattr(synIDtrans(hlID(group)), a:attr, fam)
    if code =~? pat
      return code
    endif
  endfor
  return ''
endfunction

function! s:defaults()
  let rules = copy(get(g:, 'fzf_colors', {}))
  let colors = join(map(items(filter(map(rules, 'call("s:get_color", v:val)'), '!empty(v:val)')), 'join(v:val, ":")'), ',')
  return empty(colors) ? '' : fzf#shellescape('--color='.colors)
endfunction

function! s:validate_layout(layout)
  for key in keys(a:layout)
    if index(s:layout_keys, key) < 0
      throw printf('Invalid entry in g:fzf_layout: %s (allowed: %s)%s',
            \ key, join(s:layout_keys, ', '), key == 'options' ? '. Use $FZF_DEFAULT_OPTS.' : '')
    endif
  endfor
  return a:layout
endfunction

function! s:evaluate_opts(options)
  return type(a:options) == type([]) ?
        \ join(map(copy(a:options), 'fzf#shellescape(v:val)')) : a:options
endfunction

" [name string,] [opts dict,] [fullscreen boolean]
function! fzf#wrap(...)
  let args = ['', {}, 0]
  let expects = map(copy(args), 'type(v:val)')
  let tidx = 0
  for arg in copy(a:000)
    let tidx = index(expects, type(arg), tidx)
    if tidx < 0
      throw 'Invalid arguments (expected: [name string] [opts dict] [fullscreen boolean])'
    endif
    let args[tidx] = arg
    let tidx += 1
    unlet arg
  endfor
  let [name, opts, bang] = args

  if len(name)
    let opts.name = name
  end

  " Layout: g:fzf_layout (and deprecated g:fzf_height)
  if bang
    for key in s:layout_keys
      if has_key(opts, key)
        call remove(opts, key)
      endif
    endfor
  elseif !s:has_any(opts, s:layout_keys)
    if !exists('g:fzf_layout') && exists('g:fzf_height')
      let opts.down = g:fzf_height
    else
      let opts = extend(opts, s:validate_layout(get(g:, 'fzf_layout', s:default_layout)))
    endif
  endif

  " Colors: g:fzf_colors
  let opts.options = s:defaults() .' '. s:evaluate_opts(get(opts, 'options', ''))

  " History: g:fzf_history_dir
  if len(name) && len(get(g:, 'fzf_history_dir', ''))
    let dir = s:fzf_expand(g:fzf_history_dir)
    if !isdirectory(dir)
      call mkdir(dir, 'p')
    endif
    let history = fzf#shellescape(dir.'/'.name)
    let opts.options = join(['--history', history, opts.options])
  endif

  " Action: g:fzf_action
  if !s:has_any(opts, ['sink', 'sink*'])
    let opts._action = get(g:, 'fzf_action', s:default_action)
    let opts.options .= ' --expect='.join(keys(opts._action), ',')
    function! opts.sink(lines) abort
      return s:common_sink(self._action, a:lines)
    endfunction
    let opts['sink*'] = remove(opts, 'sink')
  endif

  return opts
endfunction

function! s:use_sh()
  let [shell, shellslash, shellcmdflag, shellxquote] = [&shell, &shellslash, &shellcmdflag, &shellxquote]
  if s:is_win
    set shell=cmd.exe
    set noshellslash
    let &shellcmdflag = has('nvim') ? '/s /c' : '/c'
    let &shellxquote = has('nvim') ? '"' : '('
  else
    set shell=sh
  endif
  return [shell, shellslash, shellcmdflag, shellxquote]
endfunction

function! fzf#run(...) abort
try
  let [shell, shellslash, shellcmdflag, shellxquote] = s:use_sh()

  let dict   = exists('a:1') ? copy(a:1) : {}
  let temps  = { 'result': s:fzf_tempname() }
  let optstr = s:evaluate_opts(get(dict, 'options', ''))
  try
    let fzf_exec = s:fzf_exec()
  catch
    throw v:exception
  endtry

  if !has_key(dict, 'dir')
    let dict.dir = s:fzf_getcwd()
  endif
  if has('win32unix') && has_key(dict, 'dir')
    let dict.dir = fnamemodify(dict.dir, ':p')
  endif

  if has_key(dict, 'source')
    let source = dict.source
    let type = type(source)
    if type == 1
      let prefix = '( '.source.' )|'
    elseif type == 3
      let temps.input = s:fzf_tempname()
      call writefile(map(source, '<SID>enc_to_cp(v:val)'), temps.input)
      let prefix = (s:is_win ? 'type ' : 'cat ').fzf#shellescape(temps.input).'|'
    else
      throw 'Invalid source type'
    endif
  else
    let prefix = ''
  endif

  let prefer_tmux = get(g:, 'fzf_prefer_tmux', 0) || has_key(dict, 'tmux')
  let use_height = has_key(dict, 'down') && !has('gui_running') &&
        \ !(has('nvim') || s:is_win || has('win32unix') || s:present(dict, 'up', 'left', 'right', 'window')) &&
        \ executable('tput') && filereadable('/dev/tty')
  let has_vim8_term = has('terminal') && has('patch-8.0.995')
  let has_nvim_term = has('nvim-0.2.1') || has('nvim') && !s:is_win
  let use_term = has_nvim_term ||
    \ has_vim8_term && !has('win32unix') && (has('gui_running') || s:is_win || !use_height && s:present(dict, 'down', 'up', 'left', 'right', 'window'))
  let use_tmux = (has_key(dict, 'tmux') || (!use_height && !use_term || prefer_tmux) && !has('win32unix') && s:splittable(dict)) && s:tmux_enabled()
  if prefer_tmux && use_tmux
    let use_height = 0
    let use_term = 0
  endif
  if use_height
    let height = s:calc_size(&lines, dict.down, dict)
    let optstr .= ' --height='.height
  elseif use_term
    let optstr .= ' --no-height'
  endif
  let command = prefix.(use_tmux ? s:fzf_tmux(dict) : fzf_exec).' '.optstr.' > '.temps.result

  if use_term
    return s:execute_term(dict, command, temps)
  endif

  let lines = use_tmux ? s:execute_tmux(dict, command, temps)
                 \ : s:execute(dict, command, use_height, temps)
  call s:callback(dict, lines)
  return lines
finally
  let [&shell, &shellslash, &shellcmdflag, &shellxquote] = [shell, shellslash, shellcmdflag, shellxquote]
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
  let size = get(a:dict, 'tmux', '')
  if empty(size)
    for o in ['up', 'down', 'left', 'right']
      if s:present(a:dict, o)
        let spec = a:dict[o]
        if (o == 'up' || o == 'down') && spec[0] == '~'
          let size = '-'.o[0].s:calc_size(&lines, spec, a:dict)
        else
          " Legacy boolean option
          let size = '-'.o[0].(spec == 1 ? '' : substitute(spec, '^\~', '', ''))
        endif
        break
      endif
    endfor
  endif
  return printf('LINES=%d COLUMNS=%d %s %s %s --',
    \ &lines, &columns, fzf#shellescape(s:fzf_tmux), size, (has_key(a:dict, 'source') ? '' : '-'))
endfunction

function! s:splittable(dict)
  return s:present(a:dict, 'up', 'down') && &lines > 15 ||
        \ s:present(a:dict, 'left', 'right') && &columns > 40
endfunction

function! s:pushd(dict)
  if s:present(a:dict, 'dir')
    let cwd = s:fzf_getcwd()
    let w:fzf_pushd = {
    \   'command': haslocaldir() ? 'lcd' : (exists(':tcd') && haslocaldir(-1) ? 'tcd' : 'cd'),
    \   'origin': cwd,
    \   'bufname': bufname('')
    \ }
    execute 'lcd' s:escape(a:dict.dir)
    let cwd = s:fzf_getcwd()
    let w:fzf_pushd.dir = cwd
    let a:dict.pushd = w:fzf_pushd
    return cwd
  endif
  return ''
endfunction

augroup fzf_popd
  autocmd!
  autocmd WinEnter * call s:dopopd()
augroup END

function! s:dopopd()
  if !exists('w:fzf_pushd')
    return
  endif

  " FIXME: We temporarily change the working directory to 'dir' entry
  " of options dictionary (set to the current working directory if not given)
  " before running fzf.
  "
  " e.g. call fzf#run({'dir': '/tmp', 'source': 'ls', 'sink': 'e'})
  "
  " After processing the sink function, we have to restore the current working
  " directory. But doing so may not be desirable if the function changed the
  " working directory on purpose.
  "
  " So how can we tell if we should do it or not? A simple heuristic we use
  " here is that we change directory only if the current working directory
  " matches 'dir' entry. However, it is possible that the sink function did
  " change the directory to 'dir'. In that case, the user will have an
  " unexpected result.
  if s:fzf_getcwd() ==# w:fzf_pushd.dir && (!&autochdir || w:fzf_pushd.bufname ==# bufname(''))
    execute w:fzf_pushd.command s:escape(w:fzf_pushd.origin)
  endif
  unlet w:fzf_pushd
endfunction

function! s:xterm_launcher()
  let fmt = 'xterm -T "[fzf]" -bg "%s" -fg "%s" -geometry %dx%d+%d+%d -e bash -ic %%s'
  if has('gui_macvim')
    let fmt .= '&& osascript -e "tell application \"MacVim\" to activate"'
  endif
  return printf(fmt,
    \ escape(synIDattr(hlID("Normal"), "bg"), '#'), escape(synIDattr(hlID("Normal"), "fg"), '#'),
    \ &columns, &lines/2, getwinposx(), getwinposy())
endfunction
unlet! s:launcher
if s:is_win || has('win32unix')
  let s:launcher = '%s'
else
  let s:launcher = function('s:xterm_launcher')
endif

function! s:exit_handler(code, command, ...)
  if a:code == 130
    return 0
  elseif has('nvim') && a:code == 129
    " When deleting the terminal buffer while fzf is still running,
    " Nvim sends SIGHUP.
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

function! s:execute(dict, command, use_height, temps) abort
  call s:pushd(a:dict)
  if has('unix') && !a:use_height
    silent! !clear 2> /dev/null
  endif
  let escaped = (a:use_height || s:is_win) ? a:command : escape(substitute(a:command, '\n', '\\n', 'g'), '%#!')
  if has('gui_running')
    let Launcher = get(a:dict, 'launcher', get(g:, 'Fzf_launcher', get(g:, 'fzf_launcher', s:launcher)))
    let fmt = type(Launcher) == 2 ? call(Launcher, []) : Launcher
    if has('unix')
      let escaped = "'".substitute(escaped, "'", "'\"'\"'", 'g')."'"
    endif
    let command = printf(fmt, escaped)
  else
    let command = escaped
  endif
  if s:is_win
    let batchfile = s:fzf_tempname().'.bat'
    call writefile(s:wrap_cmds(command), batchfile)
    let command = batchfile
    let a:temps.batchfile = batchfile
    if has('nvim')
      let fzf = {}
      let fzf.dict = a:dict
      let fzf.temps = a:temps
      function! fzf.on_exit(job_id, exit_status, event) dict
        call s:pushd(self.dict)
        let lines = s:collect(self.temps)
        call s:callback(self.dict, lines)
      endfunction
      let cmd = 'start /wait cmd /c '.command
      call jobstart(cmd, fzf)
      return []
    endif
  elseif has('win32unix') && $TERM !=# 'cygwin'
    let shellscript = s:fzf_tempname()
    call writefile([command], shellscript)
    let command = 'cmd.exe /C '.fzf#shellescape('set "TERM=" & start /WAIT sh -c '.shellscript)
    let a:temps.shellscript = shellscript
  endif
  if a:use_height
    let stdin = has_key(a:dict, 'source') ? '' : '< /dev/tty'
    call system(printf('tput cup %d > /dev/tty; tput cnorm > /dev/tty; %s %s 2> /dev/tty', &lines, command, stdin))
  else
    execute 'silent !'.command
  endif
  let exit_status = v:shell_error
  redraw!
  return s:exit_handler(exit_status, command) ? s:collect(a:temps) : []
endfunction

function! s:execute_tmux(dict, command, temps) abort
  let command = a:command
  let cwd = s:pushd(a:dict)
  if len(cwd)
    " -c '#{pane_current_path}' is only available on tmux 1.9 or above
    let command = join(['cd', fzf#shellescape(cwd), '&&', command])
  endif

  call system(command)
  let exit_status = v:shell_error
  redraw!
  return s:exit_handler(exit_status, command) ? s:collect(a:temps) : []
endfunction

function! s:calc_size(max, val, dict)
  let val = substitute(a:val, '^\~', '', '')
  if val =~ '%$'
    let size = a:max * str2nr(val[:-2]) / 100
  else
    let size = min([a:max, str2nr(val)])
  endif

  let srcsz = -1
  if type(get(a:dict, 'source', 0)) == type([])
    let srcsz = len(a:dict.source)
  endif

  let opts = $FZF_DEFAULT_OPTS.' '.s:evaluate_opts(get(a:dict, 'options', ''))
  let margin = match(opts, '--inline-info\|--info[^-]\{-}inline') > match(opts, '--no-inline-info\|--info[^-]\{-}\(default\|hidden\)') ? 1 : 2
  let margin += stridx(opts, '--border') > stridx(opts, '--no-border') ? 2 : 0
  if stridx(opts, '--header') > stridx(opts, '--no-header')
    let margin += len(split(opts, "\n"))
  endif
  return srcsz >= 0 ? min([srcsz + margin, size]) : size
endfunction

function! s:getpos()
  return {'tab': tabpagenr(), 'win': winnr(), 'winid': win_getid(), 'cnt': winnr('$'), 'tcnt': tabpagenr('$')}
endfunction

function! s:split(dict)
  let directions = {
  \ 'up':    ['topleft', 'resize', &lines],
  \ 'down':  ['botright', 'resize', &lines],
  \ 'left':  ['vertical topleft', 'vertical resize', &columns],
  \ 'right': ['vertical botright', 'vertical resize', &columns] }
  let ppos = s:getpos()
  try
    if s:present(a:dict, 'window')
      if type(a:dict.window) == type({})
        if !has('nvim') && !has('patch-8.2.191')
          throw 'Vim 8.2.191 or later is required for pop-up window'
        end
        call s:popup(a:dict.window)
      else
        execute 'keepalt' a:dict.window
      endif
    elseif !s:splittable(a:dict)
      execute (tabpagenr()-1).'tabnew'
    else
      for [dir, triple] in items(directions)
        let val = get(a:dict, dir, '')
        if !empty(val)
          let [cmd, resz, max] = triple
          if (dir == 'up' || dir == 'down') && val[0] == '~'
            let sz = s:calc_size(max, val, a:dict)
          else
            let sz = s:calc_size(max, val, {})
          endif
          execute cmd sz.'new'
          execute resz sz
          return [ppos, {}]
        endif
      endfor
    endif
    return [ppos, { '&l:wfw': &l:wfw, '&l:wfh': &l:wfh }]
  finally
    setlocal winfixwidth winfixheight
  endtry
endfunction

function! s:execute_term(dict, command, temps) abort
  let winrest = winrestcmd()
  let pbuf = bufnr('')
  let [ppos, winopts] = s:split(a:dict)
  call s:use_sh()
  let b:fzf = a:dict
  let fzf = { 'buf': bufnr(''), 'pbuf': pbuf, 'ppos': ppos, 'dict': a:dict, 'temps': a:temps,
            \ 'winopts': winopts, 'winrest': winrest, 'lines': &lines,
            \ 'columns': &columns, 'command': a:command }
  function! fzf.switch_back(inplace)
    if a:inplace && bufnr('') == self.buf
      if bufexists(self.pbuf)
        execute 'keepalt b' self.pbuf
      endif
      " No other listed buffer
      if bufnr('') == self.buf
        enew
      endif
    endif
  endfunction
  function! fzf.on_exit(id, code, ...)
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
      silent! execute 'tabnext' self.ppos.tab
      silent! execute self.ppos.win.'wincmd w'
    endif

    if bufexists(self.buf)
      execute 'bd!' self.buf
    endif

    if &lines == self.lines && &columns == self.columns && s:getpos() == self.ppos
      execute self.winrest
    endif

    if !s:exit_handler(a:code, self.command, 1)
      return
    endif

    call s:pushd(self.dict)
    let lines = s:collect(self.temps)
    call s:callback(self.dict, lines)
    call self.switch_back(s:getpos() == self.ppos)
  endfunction

  try
    call s:pushd(a:dict)
    if s:is_win
      let fzf.temps.batchfile = s:fzf_tempname().'.bat'
      call writefile(s:wrap_cmds(a:command), fzf.temps.batchfile)
      let command = fzf.temps.batchfile
    else
      let command = a:command
    endif
    let command .= s:term_marker
    if has('nvim')
      call termopen(command, fzf)
    else
      if !len(&bufhidden)
        setlocal bufhidden=hide
      endif
      let fzf.buf = term_start([&shell, &shellcmdflag, command], {'curwin': 1, 'exit_cb': function(fzf.on_exit)})
      if !has('patch-8.0.1261') && !has('nvim') && !s:is_win
        call term_wait(fzf.buf, 20)
      endif
    endif
  finally
    call s:dopopd()
  endtry
  setlocal nospell bufhidden=wipe nobuflisted nonumber
  setf fzf
  startinsert
  return []
endfunction

function! s:collect(temps) abort
  try
    return filereadable(a:temps.result) ? readfile(a:temps.result) : []
  finally
    for tf in values(a:temps)
      silent! call delete(tf)
    endfor
  endtry
endfunction

function! s:callback(dict, lines) abort
  let popd = has_key(a:dict, 'pushd')
  if popd
    let w:fzf_pushd = a:dict.pushd
  endif

  try
    if has_key(a:dict, 'sink')
      for line in a:lines
        if type(a:dict.sink) == 2
          call a:dict.sink(line)
        else
          execute a:dict.sink s:escape(line)
        endif
      endfor
    endif
    if has_key(a:dict, 'sink*')
      call a:dict['sink*'](a:lines)
    endif
  catch
    if stridx(v:exception, ':E325:') < 0
      echoerr v:exception
    endif
  endtry

  " We may have opened a new window or tab
  if popd
    let w:fzf_pushd = a:dict.pushd
    call s:dopopd()
  endif
endfunction

if has('nvim')
  function s:create_popup(hl, opts) abort
    let buf = nvim_create_buf(v:false, v:true)
    let opts = extend({'relative': 'editor', 'style': 'minimal'}, a:opts)
    let border = has_key(opts, 'border') ? remove(opts, 'border') : []
    let win = nvim_open_win(buf, v:true, opts)
    call setwinvar(win, '&winhighlight', 'NormalFloat:'..a:hl)
    call setwinvar(win, '&colorcolumn', '')
    if !empty(border)
      call nvim_buf_set_lines(buf, 0, -1, v:true, border)
    endif
    return buf
  endfunction
else
  function! s:create_popup(hl, opts) abort
    let is_frame = has_key(a:opts, 'border')
    let buf = is_frame ? '' : term_start(&shell, #{hidden: 1, term_finish: 'close'})
    let id = popup_create(buf, #{
      \ line: a:opts.row,
      \ col: a:opts.col,
      \ minwidth: a:opts.width,
      \ minheight: a:opts.height,
      \ zindex: 50 - is_frame,
    \ })

    if is_frame
      call setwinvar(id, '&wincolor', a:hl)
      call setbufline(winbufnr(id), 1, a:opts.border)
      execute 'autocmd BufWipeout * ++once call popup_close('..id..')'
    else
      execute 'autocmd BufWipeout * ++once call term_sendkeys('..buf..', "exit\<CR>")'
    endif
    return winbufnr(id)
  endfunction
endif

function! s:popup(opts) abort
  " Support ambiwidth == 'double'
  let ambidouble = &ambiwidth == 'double' ? 2 : 1

  " Size and position
  let width = min([max([0, float2nr(&columns * a:opts.width)]), &columns])
  let width += width % ambidouble
  let height = min([max([0, float2nr(&lines * a:opts.height)]), &lines - has('nvim')])
  let row = float2nr(get(a:opts, 'yoffset', 0.5) * (&lines - height))
  let col = float2nr(get(a:opts, 'xoffset', 0.5) * (&columns - width))

  " Managing the differences
  let row = min([max([0, row]), &lines - has('nvim') - height])
  let col = min([max([0, col]), &columns - width])
  let row += !has('nvim')
  let col += !has('nvim')

  " Border style
  let style = tolower(get(a:opts, 'border', 'rounded'))
  if !has_key(a:opts, 'border') && !get(a:opts, 'rounded', 1)
    let style = 'sharp'
  endif

  if style =~ 'vertical\|left\|right'
    let mid = style == 'vertical' ? '│' .. repeat(' ', width - 2 * ambidouble) .. '│' :
            \ style == 'left'     ? '│' .. repeat(' ', width - 1 * ambidouble)
            \                     :        repeat(' ', width - 1 * ambidouble) .. '│'
    let border = repeat([mid], height)
    let shift = { 'row': 0, 'col': style == 'right' ? 0 : 2, 'width': style == 'vertical' ? -4 : -2, 'height': 0 }
  elseif style =~ 'horizontal\|top\|bottom'
    let hor = repeat('─', width / ambidouble)
    let mid = repeat(' ', width)
    let border = style == 'horizontal' ? [hor] + repeat([mid], height - 2) + [hor] :
               \ style == 'top'        ? [hor] + repeat([mid], height - 1)
               \                       :         repeat([mid], height - 1) + [hor]
    let shift = { 'row': style == 'bottom' ? 0 : 1, 'col': 0, 'width': 0, 'height': style == 'horizontal' ? -2 : -1 }
  else
    let edges = style == 'sharp' ? ['┌', '┐', '└', '┘'] : ['╭', '╮', '╰', '╯']
    let bar = repeat('─', width / ambidouble - 2)
    let top = edges[0] .. bar .. edges[1]
    let mid = '│' .. repeat(' ', width - 2 * ambidouble) .. '│'
    let bot = edges[2] .. bar .. edges[3]
    let border = [top] + repeat([mid], height - 2) + [bot]
    let shift = { 'row': 1, 'col': 2, 'width': -4, 'height': -2 }
  endif

  let highlight = get(a:opts, 'highlight', 'Comment')
  let frame = s:create_popup(highlight, {
    \ 'row': row, 'col': col, 'width': width, 'height': height, 'border': border
  \ })
  call s:create_popup('Normal', {
    \ 'row': row + shift.row, 'col': col + shift.col, 'width': width + shift.width, 'height': height + shift.height
  \ })
  if has('nvim')
    execute 'autocmd BufWipeout <buffer> bwipeout '..frame
  endif
endfunction

let s:default_action = {
  \ 'ctrl-t': 'tab split',
  \ 'ctrl-x': 'split',
  \ 'ctrl-v': 'vsplit' }

function! s:shortpath()
  let short = fnamemodify(getcwd(), ':~:.')
  if !has('win32unix')
    let short = pathshorten(short)
  endif
  let slash = (s:is_win && !&shellslash) ? '\' : '/'
  return empty(short) ? '~'.slash : short . (short =~ escape(slash, '\').'$' ? '' : slash)
endfunction

function! s:cmd(bang, ...) abort
  let args = copy(a:000)
  let opts = { 'options': ['--multi'] }
  if len(args) && isdirectory(expand(args[-1]))
    let opts.dir = substitute(substitute(remove(args, -1), '\\\(["'']\)', '\1', 'g'), '[/\\]*$', '/', '')
    if s:is_win && !&shellslash
      let opts.dir = substitute(opts.dir, '/', '\\', 'g')
    endif
    let prompt = opts.dir
  else
    let prompt = s:shortpath()
  endif
  let prompt = strwidth(prompt) < &columns - 20 ? prompt : '> '
  call extend(opts.options, ['--prompt', prompt])
  call extend(opts.options, args)
  call fzf#run(fzf#wrap('FZF', opts, a:bang))
endfunction

command! -nargs=* -complete=dir -bang FZF call s:cmd(<bang>0, <f-args>)

let &cpo = s:cpo_save
unlet s:cpo_save
