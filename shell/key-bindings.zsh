#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ key-bindings.zsh
#
# - $FZF_TMUX_OPTS
# - $FZF_CTRL_T_COMMAND
# - $FZF_CTRL_T_OPTS
# - $FZF_CTRL_R_OPTS
# - $FZF_ALT_C_COMMAND
# - $FZF_ALT_C_OPTS

# Key bindings
# ------------

# The code at the top and the bottom of this file is the same as in completion.zsh.
# Refer to that file for explanation.
if 'zmodload' 'zsh/parameter' 2>'/dev/null' && (( ${+options} )); then
  __fzf_key_bindings_options="options=(${(j: :)${(kv)options[@]}})"
else
  () {
    __fzf_key_bindings_options="setopt"
    'local' '__fzf_opt'
    for __fzf_opt in "${(@)${(@f)$(set -o)}%% *}"; do
      if [[ -o "$__fzf_opt" ]]; then
        __fzf_key_bindings_options+=" -o $__fzf_opt"
      else
        __fzf_key_bindings_options+=" +o $__fzf_opt"
      fi
    done
  }
fi

'emulate' 'zsh' '-o' 'no_aliases'

{

[[ -o interactive ]] || return 0

# Find the longest existing filepath from input string
# Adapted from the Fish function
__fzf_get_dir() {
  local dir="$1"

  if [[ "$dir" != "." ]]; then
    while [[ ! -d "$dir" ]]; do
      dir=${dir:h}
    done
  fi
  echo "$dir"
}

__fzf_parse_commandline() {
  autoload -Uz split-shell-arguments

  local cmdline prefix dir fzf_query
  integer cursoroffset
  local lbuffer=$LBUFFER

  local MATCH
  if [[ "$lbuffer" =~ '^-[^\s=]+=' ]]; then
    LBUFFER=${~~lbuffer#$MATCH}
    prefix=$MATCH
  fi

  # We cannot declare a local 'reply' variable or we risk overriding the callee's 'reply' variable
  local word
  () {
    local -a reply
    local REPLY REPLY2
    split-shell-arguments

    word=${reply[REPLY]}
    # We are on whitespace if REPLY is an odd number, therefore we check if we are directly after a shell word
    # by checking that the current cursor is on the first whitespace character
    if [[ $(( REPLY % 2 )) -eq 1 || $word = [[:space:]] ]]; then
      if [[ ${reply[$REPLY-1][-1]} = ${LBUFFER[-1]} ]]; then
        word=${reply[$REPLY-1]}
      else
        word=""
      fi
      (( cursoroffset = 0 ))
    else
      (( cursoroffset = ${#word} + 1 - $REPLY2 ))
    fi
  }

  # Expand variables in the path and remove one layer of quotes for __fzf_get_dir
  cmdline=${(Q)${(e)word}}

  if [[ -z "$cmdline" ]]; then
    dir="."
    fzf_query=""
  else
    dir=$(__fzf_get_dir "$cmdline")
    if [[ "$dir" = "." && "${cmdline[0,1]}" != "." ]]; then
      # if $dir is "." but cmdline is not a relative path, this means no file path found
      fzf_query=$cmdline
    else
      # Remove the longest existing directory from the cmdline, using the rest of the path as a query
      fzf_query=${${~~cmdline#$dir}#/}
    fi
  fi

  reply=("$dir" "$fzf_query" "$prefix" $cursoroffset)
}

__fzfcmd() {
  [ -n "$TMUX_PANE" ] && { [ "${FZF_TMUX:-0}" != 0 ] || [ -n "$FZF_TMUX_OPTS" ]; } &&
    echo "fzf-tmux ${FZF_TMUX_OPTS:--d${FZF_TMUX_HEIGHT:-40%}} -- " || echo "fzf"
}

# CTRL-T - Paste the selected file path(s) into the command line
fzf-file-widget() {
  local -a reply
  __fzf_parse_commandline
  local dir=$reply[1]
  local fzf_query=$reply[2]
  local prefix=$reply[3]
  local cursoroffset=$reply[4]

  local cmd=${FZF_CTRL_T_COMMAND:-"command find -L \$dir -mindepth 1 \\( -path \$dir'*/\\.*' -o -fstype 'sysfs' -o -fstype 'devfs' -o -fstype 'devtmpfs' -o -fstype 'proc' \\) -prune \
    -o -type f -print \
    -o -type d -print \
    -o -type l -print 2> /dev/null | sed 's@^\./@@'"}

  setopt localoptions pipefail no_aliases 2> /dev/null

  # Don't set FZF_DEFAULT_OPTS otherwise every invocation of the widget will expand the variable
  local local_opts="--height ${FZF_TMUX_HEIGHT:-40%} --reverse --bind=ctrl-z:ignore $FZF_DEFAULT_OPTS $FZF_CTRL_T_OPTS"

  local result=()
  local item
  # eval traps SIGINT, allowing us to check whether the result is empty
  eval "$cmd"' | FZF_DEFAULT_OPTS=$local_opts $(__fzfcmd) -m --query "$fzf_query"' | while read item; do
    result+=(${(q)item})
  done

  if (( ${#result[@]} )); then
    autoload -Uz modify-current-argument
    # Join results with a space and leave an extra space at the end
    modify-current-argument '${ARG::=${(j: :)result[@]} }'
    # The function `modify-current-argument` "tries" to retain the position of the cursor when replacing the argument
    # by moving the cursor relative to the end of the word.
    # cursoroffset contains the offset of the cursor in the current word starting from the end
    # This "offset" is used to always put the cursor at the end of the inserted text
    (( CURSOR += $cursoroffset ))
  fi

  local ret=$?
  zle reset-prompt
  return $ret
}
zle     -N   fzf-file-widget
bindkey '^T' fzf-file-widget

# ALT-C - cd into the selected directory
fzf-cd-widget() {
  local -a reply
  __fzf_parse_commandline
  local dir=$reply[1]
  local fzf_query=$reply[2]
  local prefix=$reply[3]

  local cmd=${FZF_ALT_C_COMMAND:-"command find -L \$dir -mindepth 1 \\( -path \$dir'*/\\.*' -o -fstype 'sysfs' -o -fstype 'devfs' -o -fstype 'devtmpfs' -o -fstype 'proc' \\) -prune \
    -o -type d -print 2> /dev/null | sed 's@^\./@@'"}
  setopt localoptions pipefail no_aliases 2> /dev/null
  local dir=$(eval "$cmd" | FZF_DEFAULT_OPTS="--height ${FZF_TMUX_HEIGHT:-40%} --reverse --bind=ctrl-z:ignore $FZF_DEFAULT_OPTS $FZF_ALT_C_OPTS" $(__fzfcmd) +m --query "$fzf_query")

  if [[ ! -z "$dir" ]]; then
    cd -- ${(q)dir}
    local ret=$?
    unset dir # ensure this doesn't end up appearing in prompt expansion
    zle reset-prompt
    return $ret
  else
    zle redisplay
    return 0
  fi
}
zle     -N    fzf-cd-widget
bindkey '\ec' fzf-cd-widget

# CTRL-R - Paste the selected command from history into the command line
fzf-history-widget() {
  local selected num
  setopt localoptions noglobsubst noposixbuiltins pipefail no_aliases 2> /dev/null
  selected=( $(fc -rl 1 | perl -ne 'print if !$seen{(/^\s*[0-9]+\**\s+(.*)/, $1)}++' |
    FZF_DEFAULT_OPTS="--height ${FZF_TMUX_HEIGHT:-40%} $FZF_DEFAULT_OPTS -n2..,.. --tiebreak=index --bind=ctrl-r:toggle-sort,ctrl-z:ignore $FZF_CTRL_R_OPTS --query=${(qqq)LBUFFER} +m" $(__fzfcmd)) )
  local ret=$?
  if [ -n "$selected" ]; then
    num=$selected[1]
    if [ -n "$num" ]; then
      zle vi-fetch-history -n $num
    fi
  fi
  zle reset-prompt
  return $ret
}
zle     -N   fzf-history-widget
bindkey '^R' fzf-history-widget

} always {
  eval $__fzf_key_bindings_options
  'unset' '__fzf_key_bindings_options'
}
