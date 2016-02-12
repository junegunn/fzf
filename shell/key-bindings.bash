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

__fzf_history__() {
  local out key line num cmd keyseq
  shopt -u nocaseglob nocasematch
  if [ -v _fzf_vi_mode ]; then
    out=$(
      HISTTIMEFORMAT= history |
      $(__fzfcmd) +s --tac +m -n2..,.. --tiebreak=index --bind ctrl-r:up,ctrl-s:down --expect ctrl-g)
  else
    out=$(
      HISTTIMEFORMAT= history |
      $(__fzfcmd) +s --tac +m -n2..,.. --tiebreak=index --bind ctrl-r:up,ctrl-s:down --expect ctrl-g,ctrl-o)
  fi
  key=$(head -1 <<< "$out")
  line=$(head -2 <<< "$out" | tail -1)
  num=$(sed 's/^ *\([0-9]*\)\** .*/!\1/' <<< "$line")
  if [ "$key" == "ctrl-g" ]; then
    # abort, keep existing line
    :
  else
    # accept the changes
    cmd=$(history -p ${num} 2>/dev/null)
    keyseq=""
    if [ -z "$cmd" ]; then
        # Won't work if the last command is picked
        if [[ $- =~ H ]]; then
             # Can send back ! notation and send a keysequence to escape it
             READLINE_LINE=${num}
             # expand history, end of line
             keyseq="${keyseq}${__history_expand_line}${__end_of_line}"
         else
             # Can't use shell to expand sadly, grep it out
             cmd=$(sed 's/^ *\([0-9]*\)\** *//' <<< "$line")
             READLINE_LINE=${cmd}
             READLINE_POINT=${#cmd}
         fi
    else
        READLINE_LINE=${cmd}
        READLINE_POINT=${#cmd}
    fi
    if [ "$key" == "ctrl-o" ]; then
        : # for now
        # Get current position, and last entry
        _fzf_history_ctrl_o_counter=$(sed 's/^\!//' <<< "$num")
        _fzf_history_ctrl_o_end=$(history 1 | sed 's/^ *\([0-9]*\)\** .*/\1/')
        # Get difference, and starting difference
        _fzf_history_ctrl_o_difference=$(( $_fzf_history_ctrl_o_end - $_fzf_history_ctrl_o_counter ))
        unset -v _fzf_history_ctrl_o_counter _fzf_history_ctrl_o_end
        export _fzf_history_ctrl_o_difference
        keyseq="${keyseq}\C-o"
    fi
    bind '"\C-x\C-f": "'${keyseq}'"'
  fi
}

__fzf_previous_history__() {
  if [ -v _fzf_history_scrollable ]; then
    if [ -v _fzf_history_ctrl_o_difference ]; then
      let _fzf_history_ctrl_o_difference++
    else
      _fzf_history_ctrl_o_difference=0
    fi
  fi
  bind '"\C-x\C-f": previous-history'
}
__fzf_next_history__() {
  if [ -v _fzf_history_scrollable -a -v _fzf_history_ctrl_o_difference ]; then
    if [ $_fzf_history_ctrl_o_difference -eq 0 ]; then
      unset -v _fzf_history_ctrl_o_difference
    else
      let _fzf_history_ctrl_o_difference--
    fi
  fi
  bind '"\C-x\C-f": next-history'
}

__fzf_accept_line__() {
  if [ -v _fzf_history_ctrl_o_present ]; then
    # We are inside ctrl-o, but next time we might not
    unset -v _fzf_history_ctrl_o_present
  else
    # We are not inside ctrl-o, reset counters
    unset -v _fzf_history_ctrl_o_difference
    export _fzf_history_scrollable=""
  fi
  bind '"\C-x\C-f": accept-line'
}

__fzf_ctrl_o__() {
  local keyseq

  keyseq=${__accept_line}
  export _fzf_history_ctrl_o_present=1

  if [ \! -v _fzf_history_ctrl_o_difference -a -z "$READLINE_LINE" ]; then
    # We have done ctrl-o without typing anything in, just do accept-line
    :
  else
    # We have selected back in history
    unset -v _fzf_history_scrollable
    if [[ $- =~ H ]]; then
      # Can send back ! notation and sequence to excape it
      keyseq="${keyseq}!-$(( ${_fzf_history_ctrl_o_difference} + 1 ))${__history_expand_line}${__end_of_line}"
    else
      # Can't use shell to expand sadly
      :
    fi
  fi

  bind '"\C-x\C-f": "'${keyseq}'"'

}

__use_tmux=0
__use_tmux_auto=0
if [ -n "$TMUX_PANE" ]; then
  [ ${FZF_TMUX:-1} -ne 0 -a ${LINES:-40} -gt 15 ] && __use_tmux=1
  [ $BASH_VERSINFO -gt 3 ] && __use_tmux_auto=1
fi

if [ -z "$(set -o | \grep '^vi.*on')" ]; then
  unset -v _fzf_vi_mode
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
  bind -x '"\C-x\C-h": "__fzf_history__"'
  bind '"\C-r": "\C-x\C-h\C-x\C-f"'

  bind -x '"\C-x\C-o": "__fzf_ctrl_o__"'
  bind '"\C-o": "\C-x\C-o\C-x\C-f"'
  bind -x '"\C-x'${__accept_line}'": "__fzf_accept_line__"'
  bind '"'${__accept_line}'": "\C-x'${__accept_line}'\C-x\C-f"'
  bind -x '"\C-x[A": "__fzf_previous_history__"'
  bind '"\e[A": "\C-x[A\C-x\C-f"'
  bind -x '"\C-x[B": "__fzf_next_history__"'
  bind '"\e[B": "\C-x[B\C-x\C-f"'
  export _fzf_history_scrollable=""
  trap 'export _fzf_history_scrollable=""; unset -v _fzf_history_ctrl_o_difference' INT

  # ALT-C - cd into the selected directory
  bind '"\ec": " '${__end_of_line}''${__unix_line_discard}'$(__fzf_cd__)'${__shell_expand_line}''${__redraw_current_line}''${__accept_line}'"'

  export __history_expand_line __end_of_line __accept_line
  unset -v __redraw_current_line __beginning_of_line __delete_char __backward_delete_char __unix_line_discard __kill_line __yank __yank_pop __shell_expand_line
else
  export _fzf_vi_mode=""
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
  bind -x '"\C-x\C-h": "__fzf_history__"'
  bind '"\C-r": "\C-x\C-h\C-x\C-f"'
  bind -m vi-command '"\C-r": "i\C-r"'

  # ALT-C - cd into the selected directory
  bind '"\ec": "'${__vi_command_mode}${__kill_whole_line}${__vi_insertion_mode}'$(__fzf_cd__)'${__shell_expand_line}''${__redraw_current_line}''${__accept_line}'"'
  bind -m vi-command '"\ec": "i\ec"'

  # Make end-of-line work from insert mode (like emacs mode)
  __end_of_line="${__vi_command_mode}${__end_of_line}${__vi_append_mode}"
  export __history_expand_line __end_of_line
  unset -v __shell_expand_line __redraw_current_line __vi_command_mode __vi_insertion_mode __vi_append_mode __beginning_of_line __forward_backward_delete_char __kill_whole_line __put_before __accept_line
fi

unset -v __use_tmux __use_tmux_auto

fi
