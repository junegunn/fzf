#!/bin/bash
#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/-completion.bash
#
# - $FZF_COMPLETION_TRIGGER (default: '**')
# - $FZF_COMPLETION_OPTS    (default: empty)

_fzf_orig_completion_filter() {
  sed 's/.*-F *\([^ ]*\).* \([^ ]*\)$/export _fzf_orig_completion_\2=\1;/' |
  sed 's/[^a-z0-9_= ;]/_/g'
}

_fzf_opts_completion() {
  local cur opts
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  opts="
    -x --extended
    -e --extended-exact
    -i +i
    -n --nth
    -d --delimiter
    +s --no-sort
    --tac
    -m --multi
    --no-mouse
    +c --no-color
    +2 --no-256
    --black
    --reverse
    --prompt
    -q --query
    -1 --select-1
    -0 --exit-0
    -f --filter
    --print-query
    --sync"

  if [[ ${cur} =~ ^-|\+ ]]; then
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
  fi

  return 0
}

_fzf_handle_dynamic_completion() {
  local cmd orig ret
  cmd="$1"
  shift

  orig=$(eval "echo \$_fzf_orig_completion_$cmd")
  if [ -n "$orig" ] && type "$orig" > /dev/null 2>&1; then
    $orig "$@"
  elif [ -n "$_fzf_completion_loader" ]; then
    _completion_loader "$@"
    ret=$?
    eval $(complete | \grep "\-F.* $cmd$" | _fzf_orig_completion_filter)
    source $BASH_SOURCE
    return $ret
  fi
}

_fzf_path_completion() {
  local cur base dir leftover matches trigger cmd
  cmd=$(echo ${COMP_WORDS[0]} | sed 's/[^a-z0-9_=]/_/g')
  COMPREPLY=()
  trigger=${FZF_COMPLETION_TRIGGER:-**}
  cur="${COMP_WORDS[COMP_CWORD]}"
  if [[ ${cur} == *"$trigger" ]]; then
    base=${cur:0:${#cur}-${#trigger}}
    eval base=$base

    dir="$base"
    while [ 1 ]; do
      if [ -z "$dir" -o -d "$dir" ]; then
        leftover=${base/#"$dir"}
        leftover=${leftover/#\/}
        [ "$dir" = './' ] && dir=''
        tput sc
        matches=$(find -L "$dir"* $1 2> /dev/null | fzf $FZF_COMPLETION_OPTS $2 -q "$leftover" | while read item; do
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

_fzf_list_completion() {
  local cur selected trigger cmd src
  read -r src
  cmd=$(echo ${COMP_WORDS[0]} | sed 's/[^a-z0-9_=]/_/g')
  trigger=${FZF_COMPLETION_TRIGGER:-**}
  cur="${COMP_WORDS[COMP_CWORD]}"
  if [[ ${cur} == *"$trigger" ]]; then
    cur=${cur:0:${#cur}-${#trigger}}

    tput sc
    selected=$(eval "$src | fzf $FZF_COMPLETION_OPTS $1 -q '$cur'" | tr '\n' ' ')
    selected=${selected% }
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

_fzf_all_completion() {
  _fzf_path_completion \
    "-name .git -prune -o -name .svn -prune -o -type d -print -o -type f -print -o -type l -print" \
    "-m" "" "$@"
}

_fzf_file_completion() {
  _fzf_path_completion \
    "-name .git -prune -o -name .svn -prune -o -type f -print -o -type l -print" \
    "-m" "" "$@"
}

_fzf_dir_completion() {
  _fzf_path_completion \
    "-name .git -prune -o -name .svn -prune -o -type d -print" \
    "" "/" "$@"
}

_fzf_kill_completion() {
  [ -n "${COMP_WORDS[COMP_CWORD]}" ] && return 1

  local selected
  tput sc
  selected=$(ps -ef | sed 1d | fzf -m $FZF_COMPLETION_OPTS | awk '{print $2}' | tr '\n' ' ')
  tput rc

  if [ -n "$selected" ]; then
    COMPREPLY=( "$selected" )
    return 0
  fi
}

_fzf_telnet_completion() {
  _fzf_list_completion '+m' "$@" << "EOF"
  \grep -v '^\s*\(#\|$\)' /etc/hosts | awk '{if (length($2) > 0) {print $2}}' | sort -u
EOF
}

_fzf_ssh_completion() {
  _fzf_list_completion '+m' "$@" << "EOF"
    cat <(cat ~/.ssh/config /etc/ssh/ssh_config 2> /dev/null | \grep -i ^host | \grep -v '*') <(\grep -v '^\s*\(#\|$\)' /etc/hosts) | awk '{print $2}' | sort -u
EOF
}

_fzf_env_var_completion() {
  _fzf_list_completion '-m' "$@" << "EOF"
  declare -xp | sed 's/=.*//' | sed 's/.* //'
EOF
}

_fzf_alias_completion() {
  _fzf_list_completion '-m' "$@" << "EOF"
  alias | sed 's/=.*//' | sed 's/.* //'
EOF
}

# fzf options
complete -F _fzf_opts_completion fzf

d_cmds="cd pushd rmdir"
f_cmds="
  awk cat diff diff3
  emacs ex file ftp g++ gcc gvim head hg java
  javac ld less more mvim patch perl python ruby
  sed sftp sort source tail tee uniq vi view vim wc"
a_cmds="
  basename bunzip2 bzip2 chmod chown curl cp dirname du
  find git grep gunzip gzip hg jar
  ln ls mv open rm rsync scp
  svn tar unzip zip"
x_cmds="kill ssh telnet unset unalias export"

# Preserve existing completion
if [ "$_fzf_completion_loaded" != '0.8.6-1' ]; then
  # Really wish I could use associative array but OSX comes with bash 3.2 :(
  eval $(complete | \grep '\-F' | \grep -v _fzf_ |
    \grep -E " ($(echo $d_cmds $f_cmds $a_cmds $x_cmds | sed 's/ /|/g' | sed 's/+/\\+/g'))$" | _fzf_orig_completion_filter)
  export _fzf_completion_loaded=0.8.6-1
fi

if type _completion_loader > /dev/null 2>&1; then
  _fzf_completion_loader=1
fi

# Directory
for cmd in $d_cmds; do
  complete -F _fzf_dir_completion -o nospace -o plusdirs $cmd
done

# File
for cmd in $f_cmds; do
  complete -F _fzf_file_completion -o default -o bashdefault $cmd
done

# Anything
for cmd in $a_cmds; do
  complete -F _fzf_all_completion -o default -o bashdefault $cmd
done

# Kill completion
complete -F _fzf_kill_completion -o nospace -o default -o bashdefault kill

# Host completion
complete -F _fzf_ssh_completion -o default -o bashdefault ssh
complete -F _fzf_telnet_completion -o default -o bashdefault telnet

# Environment variables / Aliases
complete -F _fzf_env_var_completion -o default -o bashdefault unset
complete -F _fzf_env_var_completion -o default -o bashdefault export
complete -F _fzf_alias_completion -o default -o bashdefault unalias

unset cmd d_cmds f_cmds a_cmds x_cmds
