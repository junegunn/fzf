# Key bindings
# ------------
# CTRL-T - Paste the selected file path(s) into the command line
__fsel() {
  command find -L . \( -path '*/\.*' -o -fstype 'dev' -o -fstype 'proc' \) -prune \
    -o -type f -print \
    -o -type d -print \
    -o -type l -print 2> /dev/null | sed 1d | cut -b3- | $(__fzfcmd) -m | while read item; do
    printf '%q ' "$item"
  done
  echo
}

__fzfcmd() {
  [ ${FZF_TMUX:-1} -eq 1 ] && echo "fzf-tmux -d${FZF_TMUX_HEIGHT:-40%}" || echo "fzf"
}

if [[ $- =~ i ]]; then

fzf-file-widget() {
  LBUFFER="${LBUFFER}$(__fsel)"
  zle redisplay
}
zle     -N   fzf-file-widget
bindkey '^T' fzf-file-widget

# ALT-C - cd into the selected directory
fzf-cd-widget() {
  cd "${$(command find -L . \( -path '*/\.*' -o -fstype 'dev' -o -fstype 'proc' \) -prune \
    -o -type d -print 2> /dev/null | sed 1d | cut -b3- | $(__fzfcmd) +m):-.}"
  zle reset-prompt
}
zle     -N    fzf-cd-widget
bindkey '\ec' fzf-cd-widget

# CTRL-R - Paste the selected command from history into the command line
fzf-history-widget() {
  local selected="$(fc -nl 1 | $(__fzfcmd) -e +s --tac +m -n2..,.. --tiebreak=index --toggle-sort=ctrl-r -q "$LBUFFER")"
  if [[ "$selected" != '' ]]; then
    LBUFFER="$(awk 'NR==1 {$1==""; print } <<< "$selected"')"
  fi
  zle redisplay
}
zle     -N   fzf-history-widget
bindkey '^R' fzf-history-widget

fi
