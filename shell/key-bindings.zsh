# Key bindings
# ------------
# CTRL-T - Paste the selected file path(s) into the command line
__fsel() {
  command find -L . \( -path '*/\.*' -o -fstype 'dev' -o -fstype 'proc' \) -prune \
    -o -type f -print \
    -o -type d -print \
    -o -type l -print 2> /dev/null | sed 1d | cut -b3- | fzf -m | while read item; do
    printf '%q ' "$item"
  done
  echo
}

fzf-current-word-widget() {
    local current
    current="${LBUFFER/* /}${RBUFFER/ */}"
    if [[ -n $current && ${current[0,1]} = '~' ]] ; then
        printf %q ${HOME}${current[2,${#current}]}
    else
        printf %q $current
    fi
}
zle -N fzf-current-word-widget

fzf-dir-at-point() {
    local current_word
    local current_dir
    current_word=$(fzf-current-word-widget)
    if [[ -d $current_word ]] ; then
        current_dir="$current_word"
    elif [[ -d ${current_word:h} ]] ; then
        current_dir=${current_word:h}
    else
        current_dir="$PWD"
    fi
    printf %q $current_dir
}

if [[ $- =~ i ]]; then

if [ -n "$TMUX_PANE" -a ${FZF_TMUX:-1} -ne 0 -a ${LINES:-40} -gt 15 ]; then
  fzf-file-widget() {
    local height
    height=${FZF_TMUX_HEIGHT:-40%}
    if [[ $height =~ %$ ]]; then
      height="-p ${height%\%}"
    else
      height="-l $height"
    fi
    tmux split-window $height "cd $(printf %q "$(fzf-dir-at-point)");zsh -c 'source ~/.fzf.zsh; tmux send-keys -t $TMUX_PANE \"\$(__fsel)\"'"
  }
else
  fzf-file-widget() {
    LBUFFER="${LBUFFER}$(__fsel)"
    zle redisplay
  }
fi
zle     -N   fzf-file-widget
bindkey '^T' fzf-file-widget

# ALT-C - cd into the selected directory
fzf-cd-widget() {
  cd "${$(command find -L . \( -path '*/\.*' -o -fstype 'dev' -o -fstype 'proc' \) -prune \
    -o -type d -print 2> /dev/null | sed 1d | cut -b3- | fzf +m):-.}"
  zle reset-prompt
}
zle     -N    fzf-cd-widget
bindkey '\ec' fzf-cd-widget

# CTRL-R - Paste the selected command from history into the command line
fzf-history-widget() {
  local selected
  if selected=$(fc -l 1 | fzf +s --tac +m -n2..,.. -q "$LBUFFER"); then
    num=$(echo "$selected" | head -1 | awk '{print $1}' | sed 's/[^0-9]//g')
    LBUFFER=!$num
    zle expand-history
  fi
  zle redisplay
}
zle     -N   fzf-history-widget
bindkey '^R' fzf-history-widget

fi

