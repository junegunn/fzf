# Key bindings
# ------------
__fzf_select__() {
  local cmd="${FZF_CTRL_T_COMMAND:-"command find -L . \\( -path '*/\\.*' -o -fstype 'dev' -o -fstype 'proc' \\) -prune \
    -o -type f -print \
    -o -type d -print \
    -o -type l -print 2> /dev/null | sed 1d | cut -b3-"}"
  eval "$cmd" | fzf -m | while read item; do
    printf '%q ' "$item"
  done
  echo
}

if [[ $- =~ i ]]; then

__fzfcmd() {
  [ ${FZF_TMUX:-1} -eq 1 ] && echo "fzf-tmux -d${FZF_TMUX_HEIGHT:-40%}" || echo "fzf"
}

__fzf_select_tmux__() {
  local height
  height=${FZF_TMUX_HEIGHT:-40%}
  if [[ $height =~ %$ ]]; then
    height="-p ${height%\%}"
  else
    height="-l $height"
  fi

  tmux split-window $height "cd $(printf %q "$PWD"); FZF_DEFAULT_OPTS=$(printf %q "$FZF_DEFAULT_OPTS") PATH=$(printf %q "$PATH") FZF_CTRL_T_COMMAND=$(printf %q "$FZF_CTRL_T_COMMAND") bash -c 'source \"${BASH_SOURCE[0]}\"; tmux send-keys -t $TMUX_PANE \"\$(__fzf_select__)\"'"
}

__fzf_select_tmux_auto__() {
  if [ ${FZF_TMUX:-1} -ne 0 -a ${LINES:-40} -gt 15 ]; then
    __fzf_select_tmux__
  else
    tmux send-keys -t $TMUX_PANE "$(__fzf_select__)"
  fi
}

__fzf_cd__() {
  local cmd dir
  cmd="${FZF_ALT_C_COMMAND:-"command find -L . \\( -path '*/\\.*' -o -fstype 'dev' -o -fstype 'proc' \\) -prune \
    -o -type d -print 2> /dev/null | sed 1d | cut -b3-"}"
  dir=$(eval "$cmd" | $(__fzfcmd) +m) && printf 'cd %q' "$dir"
}

__fzf_history__() (
  local line
  shopt -u nocaseglob nocasematch
  line=$(
    HISTTIMEFORMAT= history |
    $(__fzfcmd) +s --tac +m -n2..,.. --tiebreak=index --toggle-sort=ctrl-r |
    \grep '^ *[0-9]') &&
    if [[ $- =~ H ]]; then
      sed 's/^ *\([0-9]*\)\** .*/!\1/' <<< "$line"
    else
      sed 's/^ *\([0-9]*\)\** *//' <<< "$line"
    fi
)

__use_tmux=0
__use_tmux_auto=0
if [ -n "$TMUX_PANE" ]; then
  [ ${FZF_TMUX:-1} -ne 0 -a ${LINES:-40} -gt 15 ] && __use_tmux=1
  [ $BASH_VERSINFO -gt 3 ] && __use_tmux_auto=1
fi

if [ -z "$(set -o | \grep '^vi.*on')" ]; then
  # Required to refresh the prompt after fzf
  __redraw_current_line="\er"    # Redraw current line
  __history_expand_line="\e^"    # Expand !num history commands
  bind '"'${__redraw_current_line}'": redraw-current-line'
  bind '"'${__history_expand_line}'": history-expand-line'
  # Other variables to make reading easier
  # Movements
  __beginning_of_line="\C-a"     # Move to beginning of line
  __end_of_line="\C-e"           # Move to end of line
  # Deletions
  __delete_char="\C-d"           # Delete char under point, add to kill ring
  __backward_delete_char="\C-h"  # Delete char behind point, add to kill ring
  __unix_line_discard="\C-u"     # Delete from point to start of line, add to kill ring
  __kill_line="\C-k"             # Delete from point to end of line, add to kill ring
  # Pastes
  __yank="\C-y"                  # Yank top of kill ring
  __yank_pop="\ey"               # Rotate kill ring, then yank
  # Other
  __shell_expand_line="\e\C-e"   # Expand $() commands
  __accept_line="\C-m"           # Run the command

  # CTRL-T - Paste the selected file path into the command line
  if [ $__use_tmux_auto -eq 1 ]; then
    bind -x '"\C-t": "__fzf_select_tmux_auto__"'
  elif [ $__use_tmux -eq 1 ]; then
    bind '"\C-t": " '${__unix_line_discard}' '${__beginning_of_line}''${__kill_line}'$(__fzf_select_tmux__)'${__shell_expand_line}''${__yank}''${__beginning_of_line}''${__delete_char}''${__yank}''${__yank_pop}''${__backward_delete_char}'"'
  else
    bind '"\C-t": " '${__unix_line_discard}' '${__beginning_of_line}''${__kill_line}'$(__fzf_select__)'${__shell_expand_line}''${__yank}''${__beginning_of_line}''${__yank}''${__yank_pop}''${__backward_delete_char}''${__end_of_line}''${__redraw_current_line}' '${__backward_delete_char}'"'
  fi

  # CTRL-R - Paste the selected command from history into the command line
  bind '"\C-r": " '${__end_of_line}''${__unix_line_discard}'$(__fzf_history__)'${__shell_expand_line}''${__history_expand_line}''${__redraw_current_line}'"'

  # ALT-C - cd into the selected directory
  bind '"\ec": " '${__end_of_line}''${__unix_line_discard}'$(__fzf_cd__)'${__shell_expand_line}''${__redraw_current_line}''${__accept_line}'"'

  unset -v __redraw_current_line __history_expand_line __beginning_of_line __end_of_line __delete_char __backward_delete_char __unix_line_discard __kill_line __yank __yank_pop __shell_expand_line __accept_line
else
  # Required to refresh the prompt after fzf
  __shell_expand_line="\C-x\C-e"     # Expand $() commands
  __redraw_current_line="\C-x\C-r"   # Redraw current line
  __history_expand_line="\C-x^"      # Expand !num history commands
  bind '"'${__shell_expand_line}'": shell-expand-line'
  bind '"'${__redraw_current_line}'": redraw-current-line'
  bind '"'${__history_expand_line}'": history-expand-line'
  # Other variables to make reading easier
  # Vi-Modes
  __vi_command_mode="\e"             # Enter vi-command mode
  __vi_insertion_mode="i"            # Enter vi-inserstion mode
  __vi_append_mode="a"               # Enter vi-append mode
  # Movements
  __beginning_of_line="0"            # Move to beginning of line
  __end_of_line="$"                  # Move to end of line
  # Deletions
  __forward_backward_delete_char="x" # Delete character at point, unless EOL, then character behind point
  __kill_whole_line="dd"             # Delete entire line
  # Pastes
  __put_before="P"                   # Put the last killed text behind point, no readline equivalent function name
  # Other
  __accept_line="\C-m"               # Run the command

  # CTRL-T - Paste the selected file path into the command line
  # - FIXME: Selected items are attached to the end regardless of cursor position
  if [ $__use_tmux_auto -eq 1 ]; then
    bind -x '"\C-t": "__fzf_select_tmux_auto__"'
  elif [ $__use_tmux -eq 1 ]; then
    bind '"\C-t": "'${__vi_command_mode}${__end_of_line}${__vi_append_mode}' '${__vi_command_mode}${__kill_whole_line}${__vi_insertion_mode}'$(__fzf_select_tmux__)'${__shell_expand_line}''${__vi_command_mode}${__beginning_of_line}${__put_before}${__forward_backward_delete_char}${__end_of_line}${__vi_append_mode}'"'
  else
    bind '"\C-t": "'${__vi_command_mode}${__end_of_line}${__vi_append_mode}' '${__vi_command_mode}${__kill_whole_line}${__vi_insertion_mode}'$(__fzf_select__)'${__shell_expand_line}''${__vi_command_mode}${__beginning_of_line}${__put_before}${__forward_backward_delete_char}${__end_of_line}${__vi_append_mode}' '${__redraw_current_line}''${__vi_command_mode}${__forward_backward_delete_char}${__vi_append_mode}' "'
  fi
  bind -m vi-command '"\C-t": "i\C-t"'

  # CTRL-R - Paste the selected command from history into the command line
  bind '"\C-r": "'${__vi_command_mode}${__kill_whole_line}${__vi_insertion_mode}'$(__fzf_history__)'${__shell_expand_line}''${__history_expand_line}''${__vi_command_mode}${__end_of_line}${__vi_append_mode}''${__redraw_current_line}'"'
  bind -m vi-command '"\C-r": "i\C-r"'

  # ALT-C - cd into the selected directory
  bind '"\ec": "'${__vi_command_mode}${__kill_whole_line}${__vi_insertion_mode}'$(__fzf_cd__)'${__shell_expand_line}''${__redraw_current_line}''${__accept_line}'"'
  bind -m vi-command '"\ec": "i\ec"'

  unset -v __shell_expand_line __redraw_current_line __history_expand_line __vi_command_mode __vi_insertion_mode __vi_append_mode __beginning_of_line __end_of_line __forward_backward_delete_char __kill_whole_line __put_before __accept_line
fi

unset -v __use_tmux __use_tmux_auto

fi
