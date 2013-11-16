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

let s:exec = expand('<sfile>:h:h').'/fzf'

function! fzf#run(command, args)
  try
    let tf      = tempname()
    let prefix  = exists('g:fzf_source') ? g:fzf_source.'|' : ''
    let fzf     = executable(s:exec)     ? s:exec : 'fzf'
    let options = empty(a:args)          ? get(g:, 'fzf_options', '') : a:args
    execute "silent !".prefix.fzf.' '.options." > ".tf
    if !v:shell_error
      for line in readfile(tf)
        if !empty(line)
          execute a:command.' '.line
        endif
      endfor
    endif
  finally
    redraw!
    silent! call delete(tf)
  endtry
endfunction

command! -nargs=* FZF call fzf#run('silent e', <q-args>)

