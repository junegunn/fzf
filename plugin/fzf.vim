" Copyright (c) 2015 Junegunn Choi
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

let s:default_tmux_height = '40%'
let s:launcher = 'xterm -e bash -ic %s'
let s:fzf_go = expand('<sfile>:h:h').'/bin/fzf'
let s:fzf_rb = expand('<sfile>:h:h').'/fzf'
let s:fzf_tmux = expand('<sfile>:h:h').'/bin/fzf-tmux'

let s:cpo_save = &cpo
set cpo&vim

function! s:fzf_exec()
  if !exists('s:exec')
    if executable(s:fzf_go)
      let s:exec = s:fzf_go
    else
      let path = split(system('which fzf 2> /dev/null'), '\n')
      if !v:shell_error && !empty(path)
        let s:exec = path[0]
      elseif executable(s:fzf_rb)
        let s:exec = s:fzf_rb
      else
        call system('type fzf')
        if v:shell_error
          throw 'fzf executable not found'
        else
          let s:exec = 'fzf'
        endif
      endif
    endif
    return s:exec
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
  return substitute(a:path, ' ', '\\ ', 'g')
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

function! fzf#run(...) abort
  let dict   = exists('a:1') ? s:upgrade(a:1) : {}
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
  let split = s:tmux_enabled() && s:tmux_splittable(dict)
  let command = prefix.(split ? s:fzf_tmux(dict) : fzf_exec).' '.optstr.' > '.temps.result

  if split
    return s:execute_tmux(dict, command, temps)
  else
    return s:execute(dict, command, temps)
  endif
endfunction

function! s:fzf_tmux(dict)
  let size = ''
  for o in ['up', 'down', 'left', 'right']
    if has_key(a:dict, o)
      let size = '-'.o[0].a:dict[o]
    endif
  endfor
  return printf('LINES=%d COLUMNS=%d %s %s %s --',
    \ &lines, &columns, s:fzf_tmux, size, (has_key(a:dict, 'source') ? '' : '-'))
endfunction

function! s:tmux_splittable(dict)
  return has_key(a:dict, 'up')   ||
       \ has_key(a:dict, 'down') ||
       \ has_key(a:dict, 'left') ||
       \ has_key(a:dict, 'right')
endfunction

function! s:pushd(dict)
  if !empty(get(a:dict, 'dir', ''))
    let a:dict.prev_dir = getcwd()
    execute 'chdir '.s:escape(a:dict.dir)
    return 1
  endif
  return 0
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
    return s:callback(a:dict, a:temps)
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
  let command = a:command
  if s:pushd(a:dict)
    " -c '#{pane_current_path}' is only available on tmux 1.9 or above
    let command = 'cd '.s:escape(a:dict.dir).' && '.command
  endif

  call system(command)
  return s:callback(a:dict, a:temps)
endfunction

function! s:callback(dict, temps)
  if !filereadable(a:temps.result)
    let lines = []
  else
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
    let opts.down = get(g:, 'fzf_tmux_height', s:default_tmux_height)
  endif
  call fzf#run(extend({ 'sink': 'e', 'options': join(args) }, opts))
endfunction

command! -nargs=* -complete=dir -bang FZF call s:cmd('<bang>' == '!', <f-args>)

let &cpo = s:cpo_save
unlet s:cpo_save

