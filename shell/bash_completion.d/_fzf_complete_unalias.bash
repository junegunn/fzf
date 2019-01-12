_fzf_complete_unalias() {
  local fzf="$(__fzfcmd_complete)"
  local matches
  matches=$(
    alias | sed 's/=.*//' | sed 's/.* //' \
    | FZF_DEFAULT_OPTS="--height ${FZF_TMUX_HEIGHT} --reverse $FZF_DEFAULT_OPTS $FZF_COMPLETION_OPTS" \
      $fzf -m \
    | while read -r item; do printf "%s " "$item"; done \
  )
  matches=${matches% }
  COMPREPLY=( "${matches}" )
  printf '\e[5n'
}
complete -F _fzf_complete_unalias -o default -o bashdefault unalias
