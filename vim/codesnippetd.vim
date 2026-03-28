" codesnippetd.vim - Client for codesnippetd /pipe endpoint
" Place this file in ~/.vim/plugin/ or source it from your vimrc.
"
" Suggested keymappings (add to your vimrc):
"   nmap <Leader>cp <Plug>(codesnippetd-post)   " normal mode: send yank buffer
"   xmap <Leader>cp <Plug>(codesnippetd-post)   " visual mode: send selection

if exists('g:loaded_codesnippetd')
  finish
endif
let g:loaded_codesnippetd = 1

" Host and port of the codesnippetd server.
" Override in your vimrc before sourcing this file, e.g.:
"   let g:codesnippetd_host = '192.168.1.10'
"   let g:codesnippetd_port = 9000
let g:codesnippetd_host = get(g:, 'codesnippetd_host', 'localhost')
let g:codesnippetd_port = get(g:, 'codesnippetd_port', 8999)

" Build the /pipe endpoint URL from the current variable values.
function! s:PipeUrl() abort
  return printf('http://%s:%d/pipe', g:codesnippetd_host, g:codesnippetd_port)
endfunction

" Callback invoked when the async curl job exits.
function! s:OnExit(tmpfile, url, job, code) abort
  call delete(a:tmpfile)
  if a:code != 0
    echohl ErrorMsg
    echom printf('codesnippetd: POST failed (curl exit %d)', a:code)
    echohl None
  else
    echo 'codesnippetd: sent to ' . a:url
  endif
endfunction

" POST a string to the /pipe endpoint.
" Uses job_start() for async execution when available; falls back to
" system() otherwise.
function! s:Post(content) abort
  let l:tmpfile = tempname()
  call writefile(split(a:content, "\n", 1), l:tmpfile, 'b')

  let l:url = s:PipeUrl()
  let l:cmd = [
        \ 'curl', '-s',
        \ '-X', 'POST',
        \ '-H', 'Content-Type: application/octet-stream',
        \ '--data-binary', '@' . l:tmpfile,
        \ l:url,
        \ ]

  if has('job')
    call job_start(l:cmd, {
          \ 'exit_cb': function('s:OnExit', [l:tmpfile, l:url]),
          \ })
  else
    call system(join(map(copy(l:cmd), 'shellescape(v:val)'), ' '))
    call delete(l:tmpfile)
    if v:shell_error != 0
      echohl ErrorMsg
      echom printf('codesnippetd: POST failed (curl exit %d)', v:shell_error)
      echohl None
    else
      echo 'codesnippetd: sent to ' . l:url
    endif
  endif
endfunction

" Return the text of the last visual selection without clobbering registers.
" Uses register 'a' as a scratch area, saving and restoring its contents.
function! s:VisualSelection() abort
  let l:reg_save     = getreg('a')
  let l:regtype_save = getregtype('a')
  silent normal! gv"ay
  let l:text = getreg('a')
  call setreg('a', l:reg_save, l:regtype_save)
  return l:text
endfunction

" POST the visual selection or the yank buffer to /pipe.
" a:visual == 1: called from visual mode  -> send the selection.
" a:visual == 0: called from normal mode  -> send the unnamed register (@").
function! s:PostSelectionOrYank(visual) abort
  if a:visual
    let l:content = s:VisualSelection()
    if empty(l:content)
      echohl WarningMsg
      echom 'codesnippetd: visual selection is empty'
      echohl None
      return
    endif
  else
    let l:content = getreg('"')
    if empty(l:content)
      echohl WarningMsg
      echom 'codesnippetd: yank buffer is empty'
      echohl None
      return
    endif
  endif
  call s:Post(l:content)
endfunction

" POST lines in [line1, line2] to /pipe.
function! s:PostRange(line1, line2) abort
  let l:lines = getline(a:line1, a:line2)
  let l:content = join(l:lines, "\n")
  if empty(l:content)
    echohl WarningMsg
    echom 'codesnippetd: selection is empty'
    echohl None
    return
  endif
  call s:Post(l:content)
endfunction

" :CodesnippetdPost           (normal mode) - send the current line
" :'<,'>CodesnippetdPost      (visual mode) - send the selected lines
command! -range CodesnippetdPost call s:PostRange(<line1>, <line2>)

" <Plug>(codesnippetd-post) mappings - bind these in your vimrc:
"   nmap <Leader>cp <Plug>(codesnippetd-post)
"   xmap <Leader>cp <Plug>(codesnippetd-post)
nnoremap <silent> <Plug>(codesnippetd-post) :<C-u>call <SID>PostSelectionOrYank(0)<CR>
xnoremap <silent> <Plug>(codesnippetd-post) :<C-u>call <SID>PostSelectionOrYank(1)<CR>
