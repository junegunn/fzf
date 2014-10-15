" Copyright (c) 2014 Junegunn Choi
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

let s:min_tmux_width  = 10
let s:min_tmux_height = 3
let s:default_tmux_height = '40%'
let s:launcher = 'xterm -e bash -ic %s'
let s:fzf_rb = expand('<sfile>:h:h').'/fzf'

let s:cpo_save = &cpo
set cpo&vim

function! s:fzf_exec()
  if !exists('s:exec')
    call system('type fzf')
    if v:shell_error
      let s:exec = executable(s:fzf_rb) ? s:fzf_rb : ''
    else
      let s:exec = 'fzf'
    endif
    return s:fzf_exec()
  elseif empty(s:exec)
    unlet s:exec
    throw 'fzf executable not found'
  else
    return s:exec
  endif
endfunction

function! s:tmux_enabled()
  if has('gui_running')
    return 0
  endif

  if exists('s:tmux')
    return s:tmux
  endif

  let s:tmux = 0
  if exists('$TMUX')
    let output = system('tmux -V')
    let s:tmux = !v:shell_error && output >= 'tmux 1.7'
  endif
  return s:tmux
endfunction

function! s:shellesc(arg)
  return '"'.substitute(a:arg, '"', '\\"', 'g').'"'
endfunction

function! s:escape(path)
  return substitute(a:path, ' ', '\\ ', 'g')
endfunction

function! fzf#run(...) abort
  let dict   = exists('a:1') ? a:1 : {}
  let temps  = { 'result': tempname() }
  let optstr = get(dict, 'options', '')
  try
    let fzf_exec = s:fzf_exec()
  catch
    throw v:exception
  endtry

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
  let command = prefix.fzf_exec.' '.optstr.' > '.temps.result

  if s:tmux_enabled() && s:tmux_splittable(dict)
    return s:execute_tmux(dict, command, temps)
  else
    return s:execute(dict, command, temps)
  endif
endfunction

function! s:tmux_splittable(dict)
  return
  \ min([&columns, get(a:dict, 'tmux_width', 0)]) >= s:min_tmux_width ||
  \ min([&lines, get(a:dict, 'tmux_height', get(a:dict, 'tmux', 0))]) >= s:min_tmux_height
endfunction

function! s:pushd(dict)
  if !empty(get(a:dict, 'dir', ''))
    let a:dict.prev_dir = getcwd()
    execute 'chdir '.s:escape(a:dict.dir)
  endif
endfunction

function! s:popd(dict)
  if has_key(a:dict, 'prev_dir')
    execute 'chdir '.s:escape(remove(a:dict, 'prev_dir'))
  endif
endfunction

function! s:execute(dict, command, temps)
  call s:pushd(a:dict)
  silent !clear
  if has('gui_running')
    let launcher = get(a:dict, 'launcher', get(g:, 'fzf_launcher', s:launcher))
    let command = printf(launcher, "'".substitute(a:command, "'", "'\"'\"'", 'g')."'")
  else
    let command = a:command
  endif
  execute 'silent !'.command
  redraw!
  if v:shell_error
    " Do not print error message on exit status 1
    if v:shell_error > 1
      echohl ErrorMsg
      echo 'Error running ' . command
    endif
    return []
  else
    return s:callback(a:dict, a:temps, 0)
  endif
endfunction

function! s:env_var(name)
  if exists('$'.a:name)
    return a:name . "='". substitute(expand('$'.a:name), "'", "'\\\\''", 'g') . "' "
  else
    return ''
  endif
endfunction

function! s:execute_tmux(dict, command, temps)
  let command = s:env_var('FZF_DEFAULT_OPTS').s:env_var('FZF_DEFAULT_COMMAND').a:command
  if !empty(get(a:dict, 'dir', ''))
    let command = 'cd '.s:escape(a:dict.dir).' && '.command
  endif

  let splitopt = '-v'
  if has_key(a:dict, 'tmux_width')
    let splitopt = '-h'
    let size = a:dict.tmux_width
  else
    let size = get(a:dict, 'tmux_height', get(a:dict, 'tmux'))
  endif

  if type(size) == 1 && size =~ '%$'
    let sizeopt = '-p '.size[0:-2]
  else
    let sizeopt = '-l '.size
  endif

  let s:pane = substitute(
    \ system(
      \ printf(
        \ 'tmux split-window %s %s -P -F "#{pane_id}" %s',
        \ splitopt, sizeopt, s:shellesc(command))), '\n', '', 'g')
  let s:dict = a:dict
  let s:temps = a:temps

  augroup fzf_tmux
    autocmd!
    autocmd VimResized * nested call s:tmux_check()
  augroup END
endfunction

function! s:tmux_check()
  let panes = split(system('tmux list-panes -a -F "#{pane_id}"'), '\n')

  if index(panes, s:pane) < 0
    augroup fzf_tmux
      autocmd!
    augroup END

    call s:callback(s:dict, s:temps, 1)
    redraw
  endif
endfunction

function! s:callback(dict, temps, cd)
  if !filereadable(a:temps.result)
    let lines = []
  else
    if a:cd | call s:pushd(a:dict) | endif

    let lines = readfile(a:temps.result)
    if has_key(a:dict, 'sink')
      for line in lines
        if type(a:dict.sink) == 2
          call a:dict.sink(line)
        else
          execute a:dict.sink.' '.s:escape(line)
        endif
      endfor
    endif
  endif

  for tf in values(a:temps)
    silent! call delete(tf)
  endfor

  call s:popd(a:dict)

  return lines
endfunction

function! s:cmd(bang, ...) abort
  let args = copy(a:000)
  let opts = {}
  if len(args) > 0 && isdirectory(expand(args[-1]))
    let opts.dir = remove(args, -1)
  endif
  if !a:bang
    let opts.tmux = get(g:, 'fzf_tmux_height', s:default_tmux_height)
  endif
  call fzf#run(extend({ 'sink': 'e', 'options': join(args) }, opts))
endfunction

command! -nargs=* -complete=dir -bang FZF call s:cmd('<bang>' == '!', <f-args>)

let &cpo = s:cpo_save
unlet s:cpo_save

