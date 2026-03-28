" codesnippetd.vim - Client for codesnippetd /pipe endpoint
" Place this file in ~/.vim/plugin/ or source it from your vimrc.

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
    echo 'codesnippetd: buffer posted to ' . a:url
  endif
endfunction

" POST the current buffer's content to the /pipe endpoint.
" Uses job_start() for async execution when available; falls back to
" system() otherwise.
function! s:PostBuffer() abort
  let l:tmpfile = tempname()
  call writefile(getline(1, '$'), l:tmpfile)

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
      echo 'codesnippetd: buffer posted to ' . l:url
    endif
  endif
endfunction

command! CodesnippetdPostBuffer call s:PostBuffer()
