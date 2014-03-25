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

let s:cpo_save = &cpo
set cpo&vim

call system('type fzf')
if v:shell_error
  let s:fzf_rb = expand('<sfile>:h:h').'/fzf'
  if executable(s:fzf_rb)
    let s:exec = s:fzf_rb
  else
    echoerr 'fzf executable not found'
    finish
  endif
else
  let s:exec = 'fzf'
endif

function! s:escape(path)
  return substitute(a:path, ' ', '\\ ', 'g')
endfunction

function! fzf#run(...) abort
  let dict   = exists('a:1') ? a:1 : {}
  let temps  = [tempname()]
  let result = temps[0]
  let optstr = get(dict, 'options', '')
  let cd     = has_key(dict, 'dir')

  if has_key(dict, 'source')
    let source = dict.source
    let type = type(source)
    if type == 1
      let prefix = source.'|'
    elseif type == 3
      let input = add(temps, tempname())[-1]
      call writefile(source, input)
      let prefix = 'cat '.s:escape(input).'|'
    else
      throw 'Invalid source type'
    endif
  else
    let prefix = ''
  endif

  try
    if cd
      let cwd = getcwd()
      execute 'chdir '.s:escape(dict.dir)
    endif
    execute 'silent !'.prefix.s:exec.' '.optstr.' > '.result
    redraw!
    if v:shell_error
      return []
    endif

    let lines = readfile(result)

    if has_key(dict, 'sink')
      for line in lines
        if type(dict.sink) == 2
          call dict.sink(line)
        else
          execute dict.sink.' '.s:escape(line)
        endif
      endfor
    endif
    return lines
  finally
    if cd
      execute 'chdir '.s:escape(cwd)
    endif
    for tf in temps
      silent! call delete(tf)
    endfor
  endtry
endfunction

function! s:cmd(...)
  let args = copy(a:000)
  let opts = {}
  if len(args) > 0 && isdirectory(expand(args[-1]))
    let opts.dir = remove(args, -1)
  endif
  call fzf#run(extend({ 'sink': 'e', 'options': join(args) }, opts))
endfunction

command! -nargs=* -complete=dir FZF call s:cmd(<f-args>)

let &cpo = s:cpo_save
unlet s:cpo_save

