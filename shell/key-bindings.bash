# Key bindings
# ------------
__fsel() {
  command find -L . \( -path '*/\.*' -o -fstype 'dev' -o -fstype 'proc' \) -prune \
    -o -type f -print \
    -o -type d -print \
    -o -type l -print 2> /dev/null | sed 1d | cut -b3- | fzf -m | while read item; do
    printf '%q ' "$item"
  done
  echo
}

if [[ $- =~ i ]]; then

__fsel_tmux() {
  local height
  height=${FZF_TMUX_HEIGHT:-40%}
  if [[ $height =~ %$ ]]; then
    height="-p ${height%\%}"
  else
    height="-l $height"
  fi
  tmux split-window $height "cd $(printf %q "$PWD");bash -c 'source ~/.fzf.bash; tmux send-keys -t $TMUX_PANE \"\$(__fsel)\"'"
}

__fcd() {
  local dir
  dir=$(command find -L ${1:-.} \( -path '*/\.*' -o -fstype 'dev' -o -fstype 'proc' \) -prune \
    -o -type d -print 2> /dev/null | sed 1d | cut -b3- | fzf +m) && printf 'cd %q' "$dir"
}

__use_tmux=0
[ -n "$TMUX_PANE" -a ${FZF_TMUX:-1} -ne 0 -a ${LINES:-40} -gt 15 ] && __use_tmux=1

if [ -z "$(set -o | \grep '^vi.*on')" ]; then
  # Required to refresh the prompt after fzf
  bind '"\er": redraw-current-line'

  # CTRL-T - Paste the selected file path into the command line
  if [ $__use_tmux -eq 1 ]; then
    bind '"\C-t": " \C-u \C-a\C-k$(__fsel_tmux)\e\C-e\C-y\C-a\C-d\C-y\ey\C-h"'
  else
    bind '"\C-t": " \C-u \C-a\C-k$(__fsel)\e\C-e\C-y\C-a\C-y\ey\C-h\C-e\er \C-h"'
  fi

  # CTRL-R - Paste the selected command from history into the command line
  bind '"\C-r": " \C-e\C-u$(HISTTIMEFORMAT= history | fzf +s --tac +m -n2..,.. --toggle-sort=ctrl-r | sed \"s/ *[0-9]* *//\")\e\C-e\er"'

  # ALT-C - cd into the selected directory
  bind '"\ec": " \C-e\C-u$(__fcd)\e\C-e\er\C-m"'
else
  bind '"\C-x\C-e": shell-expand-line'
  bind '"\C-x\C-r": redraw-current-line'

  # CTRL-T - Paste the selected file path into the command line
  # - FIXME: Selected items are attached to the end regardless of cursor position
  if [ $__use_tmux -eq 1 ]; then
    bind '"\C-t": "\e$a \eddi$(__fsel_tmux)\C-x\C-e\e0P$xa"'
  else
    bind '"\C-t": "\e$a \eddi$(__fsel)\C-x\C-e\e0Px$a \C-x\C-r\exa "'
  fi
  bind -m vi-command '"\C-t": "i\C-t"'

  # CTRL-R - Paste the selected command from history into the command line
  bind '"\C-r": "\eddi$(HISTTIMEFORMAT= history | fzf +s --tac +m -n2..,.. | sed \"s/ *[0-9]* *//\")\C-x\C-e\e$a\C-x\C-r"'
  bind -m vi-command '"\C-r": "i\C-r"'

  # ALT-C - cd into the selected directory
  bind '"\ec": "\eddi$(__fcd)\C-x\C-e\C-x\C-r\C-m"'
  bind -m vi-command '"\ec": "i\ec"'
fi

unset __use_tmux

fi
