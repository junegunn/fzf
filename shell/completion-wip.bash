#!/bin/bash
#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/-completion.bash
#
# - $FZF_TMUX               (default: 0)
# - $FZF_TMUX_HEIGHT        (default: '40%')
# - $FZF_COMPLETION_TRIGGER (default: '**')
# - $FZF_COMPLETION_OPTS    (default: empty)

###########################################################

# To redraw line after fzf closes (printf '\e[5n')
bind '"\e[0n": redraw-current-line'

# __fzfcmd_complete() {
#   [ -n "$TMUX_PANE" ] && [ "${FZF_TMUX:-0}" != 0 ] && [ ${LINES:-40} -gt 15 ] &&
#     echo "fzf-tmux -d${FZF_TMUX_HEIGHT:-40%}" || echo "fzf"
# }

_fzf_orig_completion_filter() {
  awk "/-F/ && !/ _fzf/ && / $1$/"'\
        { match($0, /^(.*-F) *([^ ]*).* ([^ ]*)$/, arr); \
            if (arr[3]) {
              printf "_fzf_orig_completions[%s]=\"%s %%s %s #%s\"\n",\
                arr[3], arr[1], arr[3], arr[2] \
        }}'
}

_fzf_opts_completion() {
  local cur prev opts
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  opts="
    -x --extended
    -e --exact
    --algo
    -i +i
    -n --nth
    --with-nth
    -d --delimiter
    +s --no-sort
    --tac
    --tiebreak
    -m --multi
    --no-mouse
    --bind
    --cycle
    --no-hscroll
    --jump-labels
    --height
    --literal
    --reverse
    --margin
    --inline-info
    --prompt
    --header
    --header-lines
    --ansi
    --tabstop
    --color
    --no-bold
    --history
    --history-size
    --preview
    --preview-window
    -q --query
    -1 --select-1
    -0 --exit-0
    -f --filter
    --print-query
    --expect
    --sync"

  case "${prev}" in
  --tiebreak)
    COMPREPLY=( $(compgen -W "length begin end index" -- "$cur") )
    return 0
    ;;
  --color)
    COMPREPLY=( $(compgen -W "dark light 16 bw" -- "$cur") )
    return 0
    ;;
  --history)
    COMPREPLY=()
    return 0
    ;;
  esac

  if [[ "$cur" =~ ^-|\+ ]]; then
    COMPREPLY=( $(compgen -W "${opts}" -- "$cur") )
    return 0
  fi
__fzfcmd_complete() {
  [ -n "$TMUX_PANE" ] && [ "${FZF_TMUX:-0}" != 0 ] && [ ${LINES:-40} -gt 15 ] &&
    echo "fzf-tmux -d${FZF_TMUX_HEIGHT:-40%}" || echo "fzf"
}

  return 0
}

_fzf_completion_loader() {
  local orig ret cmd
  cmd="$1"
  if [ -n "$_fzf_orig_completion_loader" ]; then
    eval "$_fzf_orig_completion_loader $cmd"
    ret=$?
  fi

  orig="$(complete -p $cmd 2>/dev/null)"
  if [ -n "$orig" ]; then
    # Save original completion
    eval "$(_fzf_orig_completion_filter $cmd <<< $orig)"

    # Replace completion function
    printf -v def "${orig%% $cmd} -F _fzf_complete_any $cmd"
    eval "$def"
  fi
  return $ret
}

_fzf_orig_completion_loader=$(complete -Dp 2>/dev/null | sed 's/^\(.*-F\) *\([^ ]*\).*/\2/')
complete -D -F _fzf_completion_loader

_fzf_run_original_completion() {
  local cmd orig
  cmd="$1"
  orig="${_fzf_orig_completions[$cmd]##*#}"
  if [ -n "$orig" ]; then
    shift
    $orig "$@"
  fi
}

_fzf_complete_any() {
    # cmd cur prev
    cur="${COMP_WORDS[COMP_CWORD]}"
    # if [[ "$cur" == "" ]]; then
      # set -- $1 "" $3
      # COMP_LINE=${COMP_LINE:0:$COMP_POINT-2}
      COMP_LINE=${COMP_LINE:0:$COMP_POINT}
      _fzf_run_original_completion $1 "$@"

      # if [ ${#COMPREPLY[@]} -lt 100 ]; then
	#       return 0
      # fi

      # echo $words
      matches=$(printf '%s\n' "${COMPREPLY[@]}" | awk '!a[$0]++' | FZF_DEFAULT_OPTS="--height ${FZF_TMUX_HEIGHT:-40%} --reverse $FZF_DEFAULT_OPTS $FZF_COMPLETION_OPTS" fzf -1 | while read -r item; do
        printf -- "${item}"
      done)
      matches=${matches% }
      printf '\e[5n'
      if [ -n "$matches" ]; then
        COMPREPLY=( "$matches" )
        return 0
      else
        # COMPREPLY=( "$cur" )
        COMPREPLY=()
      fi
    # else
      # _fzf_run_original_completion $1 "$@"
    # fi
}

# declare -A _fzf_completions _fzf_orig_completions
declare -A _fzf_orig_completions

# fzf options
complete -o default -F _fzf_opts_completion fzf

# eval "$(complete | _fzf_orig_completion_filter "${all//[[:space:]]/|}" )"
eval "$(complete -p | _fzf_orig_completion_filter ".*" )"
for orig in "${_fzf_orig_completions[@]%%*#}"; do
  # # Replace completion function
  printf -v def "$orig" _fzf_complete_any
  eval "$def"
done

# unset cmd d_cmds a_cmds x_cmds all all_cmds
unset orig
