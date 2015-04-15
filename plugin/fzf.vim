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
let s:default_split_style = 'e'
let s:tab_char = 'ctrl-t'
let s:split_char = 'ctrl-x'
let s:vsplit_char = 'ctrl-v'

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
  if has('nvim') && bufexists('[FZF]')
    echohl WarningMsg
    echomsg 'FZF is already running!'
    echohl NONE
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

  try
    if tmux
      return s:execute_tmux(dict, command, temps)
    elseif has('nvim')
      return s:execute_term(dict, command, temps)
    else
      return s:execute(dict, command, temps)
    endif
  finally
    call s:popd(dict)
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
      let size = '-'.o[0].(a:dict[o] == 1 ? '' : a:dict[o])
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
    execute 'chdir '.s:escape(a:dict.dir)
    let a:dict.dir = getcwd()
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
  silent! !clear 2> /dev/null
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

function! s:execute_tmux(dict, command, temps)
  let command = a:command
  if s:pushd(a:dict)
    " -c '#{pane_current_path}' is only available on tmux 1.9 or above
    let command = 'cd '.s:escape(a:dict.dir).' && '.command
  endif

  call system(command)
  return s:callback(a:dict, a:temps)
endfunction

function! s:calc_size(max, val)
  if a:val =~ '%$'
    return a:max * str2nr(a:val[:-2]) / 100
  else
    return min([a:max, a:val])
  endif
endfunction

function! s:split(dict)
  let directions = {
  \ 'up':    ['topleft', 'resize', &lines],
  \ 'down':  ['botright', 'resize', &lines],
  \ 'left':  ['vertical topleft', 'vertical resize', &columns],
  \ 'right': ['vertical botright', 'vertical resize', &columns] }
  let s:ptab = tabpagenr()
  try
    for [dir, triple] in items(directions)
      let val = get(a:dict, dir, '')
      if !empty(val)
        let [cmd, resz, max] = triple
        let sz = s:calc_size(max, val)
        execute cmd sz.'new'
        execute resz sz
        return
      endif
    endfor
    if s:present(a:dict, 'window')
      execute a:dict.window
    else
      tabnew
    endif
  finally
    setlocal winfixwidth winfixheight
  endtry
endfunction

function! s:execute_term(dict, command, temps)
  call s:split(a:dict)
  call s:pushd(a:dict)

  let fzf = { 'buf': bufnr('%'), 'dict': a:dict, 'temps': a:temps }
  function! fzf.on_exit(id, code)
    let tab = tabpagenr()
    execute 'bd!' self.buf
    if s:ptab == tab
      wincmd p
    endif
    call s:pushd(self.dict)
    try
      call s:callback(self.dict, self.temps)
    finally
      call s:popd(self.dict)
    endtry
  endfunction

  call termopen(a:command, fzf)
  silent file [FZF]
  startinsert
  return []
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

  return lines
endfunction

function! s:cmd_callback(lines) abort
  if empty(a:lines)
    return
  endif
  let key = remove(a:lines, 0)
  if     key == get(g:, 'fzf_tab_char', s:tab_char)             | let cmd = 'tabedit'
  elseif key == get(g:, 'fzf_split_char', s:split_char)         | let cmd = 'split'
  elseif key == get(g:, 'fzf_vsplit_char', s:vsplit_char)       | let cmd = 'vsplit'
  else                                                          | let cmd = get(g:, 'fzf_default_split_style', s:default_split_style)
  endif
  for item in a:lines
    execute cmd s:escape(item)
  endfor
endfunction

function! s:cmd(bang, ...) abort
  let chars = [ get(g:, 'fzf_tab_char', s:tab_char), get(g:, 'fzf_split_char', s:split_char), get(g:, 'fzf_vsplit_char', s:vsplit_char) ]
  let args = extend(['--expect='.join(chars,',')], a:000)
  let opts = {}
  if len(args) > 0 && isdirectory(expand(args[-1]))
    let opts.dir = remove(args, -1)
  endif
  if !a:bang
    let opts.down = get(g:, 'fzf_tmux_height', s:default_tmux_height)
  endif
  call fzf#run(extend({'options': join(args), 'sink*': function('<sid>cmd_callback')}, opts))
endfunction

command! -nargs=* -complete=dir -bang FZF call s:cmd(<bang>0, <f-args>)

let &cpo = s:cpo_save
unlet s:cpo_save

