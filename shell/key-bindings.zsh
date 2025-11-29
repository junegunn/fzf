#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ key-bindings.zsh
#
# - $FZF_TMUX_OPTS
# - $FZF_CTRL_T_COMMAND
# - $FZF_CTRL_T_OPTS
# - $FZF_CTRL_R_COMMAND
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

'builtin' 'emulate' 'zsh' && 'builtin' 'setopt' 'no_aliases'

{
if [[ -o interactive ]]; then

#----BEGIN INCLUDE common.sh
# NOTE: Do not directly edit this section, which is copied from "common.sh".
# To modify it, one can edit "common.sh" and run "./update.sh" to apply
# the changes. See code comments in "common.sh" for the implementation details.

__fzf_defaults() {
  printf '%s\n' "--height ${FZF_TMUX_HEIGHT:-40%} --min-height 20+ --bind=ctrl-z:ignore $1"
  command cat "${FZF_DEFAULT_OPTS_FILE-}" 2> /dev/null
  printf '%s\n' "${FZF_DEFAULT_OPTS-} $2"
}

__fzf_exec_awk() {
  if [[ -z ${__fzf_awk-} ]]; then
    __fzf_awk=awk
    if [[ $OSTYPE == solaris* && -x /usr/xpg4/bin/awk ]]; then
      __fzf_awk=/usr/xpg4/bin/awk
    elif command -v mawk > /dev/null 2>&1; then
      local n x y z d
      IFS=' .' read -r n x y z d <<< $(command mawk -W version 2> /dev/null)
      [[ $n == mawk ]] &&
        (((x * 1000 + y) * 1000 + z >= 1003004)) 2> /dev/null &&
        ((d >= 20230302)) 2> /dev/null &&
        __fzf_awk=mawk
    fi
  fi
  LC_ALL=C exec "$__fzf_awk" "$@"
}
#----END INCLUDE

# CTRL-T - Paste the selected file path(s) into the command line
__fzf_select() {
  setopt localoptions pipefail no_aliases 2> /dev/null
  local item
  FZF_DEFAULT_COMMAND=${FZF_CTRL_T_COMMAND:-} \
  FZF_DEFAULT_OPTS=$(__fzf_defaults "--reverse --walker=file,dir,follow,hidden --scheme=path" "${FZF_CTRL_T_OPTS-} -m") \
  FZF_DEFAULT_OPTS_FILE='' $(__fzfcmd) "$@" < /dev/tty | while read -r item; do
    echo -n -E "${(q)item} "
  done
  local ret=$?
  echo
  return $ret
}

__fzfcmd() {
  [ -n "${TMUX_PANE-}" ] && { [ "${FZF_TMUX:-0}" != 0 ] || [ -n "${FZF_TMUX_OPTS-}" ]; } &&
    echo "fzf-tmux ${FZF_TMUX_OPTS:--d${FZF_TMUX_HEIGHT:-40%}} -- " || echo "fzf"
}

fzf-file-widget() {
  LBUFFER="${LBUFFER}$(__fzf_select)"
  local ret=$?
  zle reset-prompt
  return $ret
}
if [[ "${FZF_CTRL_T_COMMAND-x}" != "" ]]; then
  zle     -N            fzf-file-widget
  bindkey -M emacs '^T' fzf-file-widget
  bindkey -M vicmd '^T' fzf-file-widget
  bindkey -M viins '^T' fzf-file-widget
fi

# ALT-C - cd into the selected directory
fzf-cd-widget() {
  setopt localoptions pipefail no_aliases 2> /dev/null
  local dir="$(
    FZF_DEFAULT_COMMAND=${FZF_ALT_C_COMMAND:-} \
    FZF_DEFAULT_OPTS=$(__fzf_defaults "--reverse --walker=dir,follow,hidden --scheme=path" "${FZF_ALT_C_OPTS-} +m") \
    FZF_DEFAULT_OPTS_FILE='' $(__fzfcmd) < /dev/tty)"
  if [[ -z "$dir" ]]; then
    zle redisplay
    return 0
  fi
  zle push-line # Clear buffer. Auto-restored on next prompt.
  BUFFER="builtin cd -- ${(q)dir:a}"
  zle accept-line
  local ret=$?
  unset dir # ensure this doesn't end up appearing in prompt expansion
  zle reset-prompt
  return $ret
}
if [[ "${FZF_ALT_C_COMMAND-x}" != "" ]]; then
  zle     -N             fzf-cd-widget
  bindkey -M emacs '\ec' fzf-cd-widget
  bindkey -M vicmd '\ec' fzf-cd-widget
  bindkey -M viins '\ec' fzf-cd-widget
fi

# CTRL-R - Paste the selected command from history into the command line
fzf-history-widget() {
  local selected extracted_with_perl=0
  setopt localoptions noglobsubst noposixbuiltins pipefail no_aliases no_glob no_ksharrays extendedglob 2> /dev/null
  # Ensure the module is loaded if not already, and the required features, such
  # as the associative 'history' array, which maps event numbers to full history
  # lines, are set. Also, make sure Perl is installed for multi-line output.
  if zmodload -F zsh/parameter p:{commands,history} 2>/dev/null && (( ${+commands[perl]} )); then
    selected="$(printf '%s\t%s\000' "${(kv)history[@]}" |
      perl -0 -ne 'if (!$seen{(/^\s*[0-9]+\**\t(.*)/s, $1)}++) { s/\n/\n\t/g; print; }' |
      FZF_DEFAULT_OPTS=$(__fzf_defaults "" "-n2..,.. --scheme=history --bind=ctrl-r:toggle-sort,alt-r:toggle-raw --wrap-sign '\t↳ ' --highlight-line --multi ${FZF_CTRL_R_OPTS-} --query=${(qqq)LBUFFER} --read0") \
      FZF_DEFAULT_OPTS_FILE='' $(__fzfcmd))"
      extracted_with_perl=1
  else
    selected="$(fc -rl 1 | __fzf_exec_awk '{ cmd=$0; sub(/^[ \t]*[0-9]+\**[ \t]+/, "", cmd); if (!seen[cmd]++) print $0 }' |
      FZF_DEFAULT_OPTS=$(__fzf_defaults "" "-n2..,.. --scheme=history --bind=ctrl-r:toggle-sort,alt-r:toggle-raw --wrap-sign '\t↳ ' --highlight-line --multi ${FZF_CTRL_R_OPTS-} --query=${(qqq)LBUFFER}") \
      FZF_DEFAULT_OPTS_FILE='' $(__fzfcmd))"
  fi
  local ret=$?
  local -a cmds
  # Avoid leaking auto assigned values when using backreferences '(#b)'
  local -a mbegin mend match
  if [ -n "$selected" ]; then
    # Heuristic to check if the selected value is from history or a custom query
    if ((( extracted_with_perl )) && [[ $selected == <->$'\t'* ]]) ||
    ((( ! extracted_with_perl )) && [[ $selected == [[:blank:]]#<->(  |\* )* ]]); then
      # Split at newlines
      for line in ${(ps:\n:)selected}; do
        if (( extracted_with_perl )); then
          if [[ $line == (#b)(<->)(#B)$'\t'* ]]; then
            (( ${+history[${match[1]}]} )) && cmds+=("${history[${match[1]}]}")
          fi
        elif [[ $line == [[:blank:]]#(#b)(<->)(#B)(  |\* )* ]]; then
          # Avoid $history array: lags behind 'fc' on foreign commands (*)
          # https://zsh.org/mla/users/2024/msg00692.html
          # Push BUFFER onto stack; fetch and save history entry from BUFFER; restore
          zle .push-line
          zle vi-fetch-history -n ${match[1]}
          (( ${#BUFFER} )) && cmds+=("${BUFFER}")
          BUFFER=""
          zle .get-line
        fi
      done
      if (( ${#cmds[@]} )); then
        # Join by newline after stripping trailing newlines from each command
        BUFFER="${(pj:\n:)${(@)cmds%%$'\n'#}}"
        CURSOR=${#BUFFER}
      fi
    else # selected is a custom query, not from history
      LBUFFER="$selected"
    fi
  fi
  zle reset-prompt
  return $ret
}
if [[ ${FZF_CTRL_R_COMMAND-x} != "" ]]; then
  if [[ -n ${FZF_CTRL_R_COMMAND-} ]]; then
    echo "warning: FZF_CTRL_R_COMMAND is set to a custom command, but custom commands are not yet supported for CTRL-R" >&2
  fi
  zle     -N            fzf-history-widget
  bindkey -M emacs '^R' fzf-history-widget
  bindkey -M vicmd '^R' fzf-history-widget
  bindkey -M viins '^R' fzf-history-widget
fi
fi

} always {
  eval $__fzf_key_bindings_options
  'unset' '__fzf_key_bindings_options'
}
