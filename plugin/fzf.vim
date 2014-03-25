" Copyright (c) 2013 Junegunn Choi
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

function! fzf#run(command, ...)
  let cwd = getcwd()
  try
    let args = copy(a:000)
    if len(args) > 0 && isdirectory(expand(args[-1]))
      let dir = remove(args, -1)
      execute 'chdir '.s:escape(dir)
    endif
    let argstr  = join(args)
    let tf      = tempname()
    let prefix  = exists('g:fzf_source') ? g:fzf_source.'|' : ''
    let options = empty(argstr)          ? get(g:, 'fzf_options', '') : argstr
    execute 'silent !'.prefix.s:exec.' '.options.' > '.tf
    if !v:shell_error
      for line in readfile(tf)
        if !empty(line)
          execute a:command.' '.s:escape(line)
        endif
      endfor
    endif
  finally
    execute 'chdir '.s:escape(cwd)
    redraw!
    silent! call delete(tf)
  endtry
endfunction

command! -nargs=* -complete=dir FZF call fzf#run('silent e', <f-args>)

