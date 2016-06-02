#!/bin/bash
#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/-completion.bash
#
# - $FZF_TMUX               (default: 1)
# - $FZF_TMUX_HEIGHT        (default: '40%')
# - $FZF_COMPLETION_TRIGGER (default: '**')
# - $FZF_COMPLETION_OPTS    (default: empty)

# To use custom commands instead of find, override _fzf_compgen_{path,dir}
if ! declare -f _fzf_compgen_path > /dev/null; then
  _fzf_compgen_path() {
    echo "$1"
    \find -L "$1" \
      -name .git -prune -o -name .svn -prune -o \( -type d -o -type f -o -type l \) \
      -a -not -path "$1" -print 2> /dev/null | sed 's@^\./@@'
  }
fi

if ! declare -f _fzf_compgen_dir > /dev/null; then
  _fzf_compgen_dir() {
    \find -L "$1" \
      -name .git -prune -o -name .svn -prune -o -type d \
      -a -not -path "$1" -print 2> /dev/null | sed 's@^\./@@'
  }
fi

###########################################################

_fzf_orig_completion_filter() {
  sed 's/^\(.*-F\) *\([^ ]*\).* \([^ ]*\)$/export _fzf_orig_completion_\3="\1 %s \3 #\2";/' |
  awk -F= '{gsub(/[^a-z0-9_= ;]/, "_", $1); print $1"="$2}'
}

_fzf_opts_completion() {
  local cur prev opts
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  opts="
    -x --extended
    -e --exact
    -i +i
    -n --nth
    -d --delimiter
    +s --no-sort
    --tac
    --tiebreak
    --bind
    -m --multi
    --no-mouse
    --color
    --black
    --reverse
    --no-hscroll
    --inline-info
    --prompt
    -q --query
    -1 --select-1
    -0 --exit-0
    -f --filter
    --print-query
    --expect
    --toggle-sort
    --sync
    --cycle
    --history
    --history-size
    --header
    --header-lines
    --margin"

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

  return 0
}

_fzf_handle_dynamic_completion() {
  local cmd orig_var orig ret orig_cmd
  cmd="$1"
  shift
  orig_cmd="$1"
  orig_var="_fzf_orig_completion_$cmd"
  orig="${!orig_var##*#}"
  if [ -n "$orig" ] && type "$orig" > /dev/null 2>&1; then
    $orig "$@"
  elif [ -n "$_fzf_completion_loader" ]; then
    _completion_loader "$@"
    ret=$?
    eval "$(complete | \grep "\-F.* $orig_cmd$" | _fzf_orig_completion_filter)"
    source "${BASH_SOURCE[0]}"
    return $ret
  fi
}

__fzf_generic_path_completion() {
  local cur base dir leftover matches trigger cmd fzf
  [ "${FZF_TMUX:-1}" != 0 ] && fzf="fzf-tmux -d ${FZF_TMUX_HEIGHT:-40%}" || fzf="fzf"
  cmd=$(echo "${COMP_WORDS[0]}" | sed 's/[^a-z0-9_=]/_/g')
  COMPREPLY=()
  trigger=${FZF_COMPLETION_TRIGGER-'**'}
  cur="${COMP_WORDS[COMP_CWORD]}"
  if [[ "$cur" == *"$trigger" ]]; then
    base=${cur:0:${#cur}-${#trigger}}
    eval "base=$base"

    dir="$base"
    while true; do
      if [ -z "$dir" ] || [ -d "$dir" ]; then
        leftover=${base/#"$dir"}
        leftover=${leftover/#\/}
        [ -z "$dir" ] && dir='.'
        [ "$dir" != "/" ] && dir="${dir/%\//}"
        tput sc
        matches=$(eval "$1 $(printf %q "$dir")" | $fzf $FZF_COMPLETION_OPTS $2 -q "$leftover" | while read -r item; do
          printf "%q$3 " "$item"
        done)
        matches=${matches% }
        if [ -n "$matches" ]; then
          COMPREPLY=( "$matches" )
        else
          COMPREPLY=( "$cur" )
        fi
        tput rc
        return 0
      fi
      dir=$(dirname "$dir")
      [[ "$dir" =~ /$ ]] || dir="$dir"/
    done
  else
    shift
    shift
    shift
    _fzf_handle_dynamic_completion "$cmd" "$@"
  fi
}

_fzf_complete() {
  local cur selected trigger cmd fzf post
  post="$(caller 0 | awk '{print $2}')_post"
  type -t "$post" > /dev/null 2>&1 || post=cat
  [ "${FZF_TMUX:-1}" != 0 ] && fzf="fzf-tmux -d ${FZF_TMUX_HEIGHT:-40%}" || fzf="fzf"

  cmd=$(echo "${COMP_WORDS[0]}" | sed 's/[^a-z0-9_=]/_/g')
  trigger=${FZF_COMPLETION_TRIGGER-'**'}
  cur="${COMP_WORDS[COMP_CWORD]}"
  if [[ "$cur" == *"$trigger" ]]; then
    cur=${cur:0:${#cur}-${#trigger}}

    tput sc
    selected=$(cat | $fzf $FZF_COMPLETION_OPTS $1 -q "$cur" | $post | tr '\n' ' ')
    selected=${selected% } # Strip trailing space not to repeat "-o nospace"
    tput rc

    if [ -n "$selected" ]; then
      COMPREPLY=("$selected")
      return 0
    fi
  else
    shift
    _fzf_handle_dynamic_completion "$cmd" "$@"
  fi
}

_fzf_path_completion() {
  __fzf_generic_path_completion _fzf_compgen_path "-m" "" "$@"
}

# Deprecated. No file only completion.
_fzf_file_completion() {
  _fzf_path_completion "$@"
}

_fzf_dir_completion() {
  __fzf_generic_path_completion _fzf_compgen_dir "" "/" "$@"
}

_fzf_complete_kill() {
  [ -n "${COMP_WORDS[COMP_CWORD]}" ] && return 1

  local selected fzf
  [ "${FZF_TMUX:-1}" != 0 ] && fzf="fzf-tmux -d ${FZF_TMUX_HEIGHT:-40%}" || fzf="fzf"
  tput sc
  selected=$(ps -ef | sed 1d | $fzf -m $FZF_COMPLETION_OPTS | awk '{print $2}' | tr '\n' ' ')
  tput rc

  if [ -n "$selected" ]; then
    COMPREPLY=( "$selected" )
    return 0
  fi
}

_fzf_complete_telnet() {
  _fzf_complete '+m' "$@" < <(
    \grep -v '^\s*\(#\|$\)' /etc/hosts | \grep -Fv '0.0.0.0' |
        awk '{if (length($2) > 0) {print $2}}' | sort -u
  )
}

_fzf_complete_ssh() {
  _fzf_complete '+m' "$@" < <(
    cat <(cat ~/.ssh/config /etc/ssh/ssh_config 2> /dev/null | \grep -i '^host' | \grep -v '*') \
        <(\grep -oE '^[^ ]+' ~/.ssh/known_hosts | tr ',' '\n' | awk '{ print $1 " " $1 }') \
        <(\grep -v '^\s*\(#\|$\)' /etc/hosts | \grep -Fv '0.0.0.0') |
        awk '{if (length($2) > 0) {print $2}}' | sort -u
  )
}

_fzf_complete_unset() {
  _fzf_complete '-m' "$@" < <(
    declare -xp | sed 's/=.*//' | sed 's/.* //'
  )
}

_fzf_complete_export() {
  _fzf_complete '-m' "$@" < <(
    declare -xp | sed 's/=.*//' | sed 's/.* //'
  )
}

_fzf_complete_unalias() {
  _fzf_complete '-m' "$@" < <(
    alias | sed 's/=.*//' | sed 's/.* //'
  )
}

# fzf options
complete -o default -F _fzf_opts_completion fzf

d_cmds="${FZF_COMPLETION_DIR_COMMANDS:-cd pushd rmdir}"
a_cmds="
  awk cat diff diff3
  emacs emacsclient ex file ftp g++ gcc gvim head hg java
  javac ld less more mvim nvim patch perl python ruby
  sed sftp sort source tail tee uniq vi view vim wc xdg-open
  basename bunzip2 bzip2 chmod chown curl cp dirname du
  find git grep gunzip gzip hg jar
  ln ls mv open rm rsync scp
  svn tar unzip zip"
x_cmds="kill ssh telnet unset unalias export"

# Preserve existing completion
if [ "$_fzf_completion_loaded" != '0.11.3' ]; then
  # Really wish I could use associative array but OSX comes with bash 3.2 :(
  eval $(complete | \grep '\-F' | \grep -v _fzf_ |
    \grep -E " ($(echo $d_cmds $a_cmds $x_cmds | sed 's/ /|/g' | sed 's/+/\\+/g'))$" | _fzf_orig_completion_filter)
  export _fzf_completion_loaded=0.11.3
fi

if type _completion_loader > /dev/null 2>&1; then
  _fzf_completion_loader=1
fi

_fzf_defc() {
  local cmd func opts orig_var orig def
  cmd="$1"
  func="$2"
  opts="$3"
  orig_var="_fzf_orig_completion_$cmd"
  orig="${!orig_var}"
  if [ -n "$orig" ]; then
    printf -v def "$orig" "$func"
    eval "$def"
  else
    complete -F "$func" $opts "$cmd"
  fi
}

# Anything
for cmd in $a_cmds; do
  _fzf_defc "$cmd" _fzf_path_completion "-o default -o bashdefault"
done

# Directory
for cmd in $d_cmds; do
  _fzf_defc "$cmd" _fzf_dir_completion "-o nospace -o plusdirs"
done

unset _fzf_defc

# Kill completion
complete -F _fzf_complete_kill -o nospace -o default -o bashdefault kill

# Host completion
complete -F _fzf_complete_ssh -o default -o bashdefault ssh
complete -F _fzf_complete_telnet -o default -o bashdefault telnet

# Environment variables / Aliases
complete -F _fzf_complete_unset -o default -o bashdefault unset
complete -F _fzf_complete_export -o default -o bashdefault export
complete -F _fzf_complete_unalias -o default -o bashdefault unalias

unset cmd d_cmds a_cmds x_cmds
