#!/bin/bash
#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/-completion.bash
#
# - $FZF_COMPLETION_TRIGGER (default: '**')
# - $FZF_COMPLETION_OPTS    (default: empty)

_fzf_opts_completion() {
  local cur prev opts
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  opts="-m --multi -x --extended -s --sort +s +i +c --no-color"

  case "${prev}" in
  --sort|-s)
    COMPREPLY=( $(compgen -W "$(seq 2000 1000 10000)" -- ${cur}) )
    return 0
    ;;
  esac

  if [[ ${cur} =~ ^-|\+ ]]; then
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
  fi

  return 0
}

_fzf_generic_completion() {
  local cur base dir leftover matches
  COMPREPLY=()
  FZF_COMPLETION_TRIGGER=${FZF_COMPLETION_TRIGGER:-**}
  cur="${COMP_WORDS[COMP_CWORD]}"
  if [[ ${cur} == *"$FZF_COMPLETION_TRIGGER" ]]; then
    base=${cur:0:${#cur}-${#FZF_COMPLETION_TRIGGER}}
    eval base=$base

    dir="$base"
    while [ 1 ]; do
      if [ -z "$dir" -o -d "$dir" ]; then
        leftover=${base/#"$dir"}
        leftover=${leftover/#\/}
        [ "$dir" = './' ] && dir=''
        tput sc
        matches=$(find "$dir"* $1 2> /dev/null | fzf $FZF_COMPLETION_OPTS $2 -q "$leftover" | while read item; do
          printf '%q ' "$item"
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
  fi
}

_fzf_all_completion() {
  _fzf_generic_completion \
    "-name .git -prune -o -name .svn -prune -o -type d -print -o -type f -print -o -type l -print" \
    "-m"
}

_fzf_file_completion() {
  _fzf_generic_completion \
    "-name .git -prune -o -name .svn -prune -o -type f -print -o -type l -print" \
    "-m"
}

_fzf_dir_completion() {
  _fzf_generic_completion \
    "-name .git -prune -o -name .svn -prune -o -type d -print" \
    ""
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
  local cur prev selected
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  [[ "$cur" =~ ^- || "$prev" =~ ^- ]] && return 1

  tput sc
  selected=$(grep -v '^\s*\(#\|$\)' /etc/hosts | awk '{print $2}' | sort -u | fzf $FZF_COMPLETION_OPTS -q "$cur")
  tput rc

  if [ -n "$selected" ]; then
    COMPREPLY=("$selected")
    return 0
  fi
}

_fzf_ssh_completion() {
  local cur prev selected
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  [[ "$cur" =~ ^- || "$prev" =~ ^- ]] && return 1

  tput sc
  selected=$(cat \
    <(cat ~/.ssh/config /etc/ssh/ssh_config 2> /dev/null | grep -i ^host) \
    <(grep -v '^\s*\(#\|$\)' /etc/hosts) | \
    awk '{print $2}' | sort -u | fzf $FZF_COMPLETION_OPTS -q "$cur")
  tput rc

  if [ -n "$selected" ]; then
    COMPREPLY=("$selected")
    return 0
  fi
}

complete -F _fzf_opts_completion fzf

# Directory
for cmd in "cd pushd rmdir"; do
  complete -F _fzf_dir_completion -o default -o bashdefault $cmd
done

# File
for cmd in "
  awk cat diff diff3
  emacs ex file ftp g++ gcc gvim head hg java
  javac ld less more mvim patch perl python ruby
  sed sftp sort source tail tee uniq vi view vim wc"; do
  complete -F _fzf_file_completion -o default -o bashdefault $cmd
done

# Anything
for cmd in "
  basename bunzip2 bzip2 chmod chown curl cp dirname du
  find git grep gunzip gzip hg jar
  ln ls mv open rm rsync scp
  svn tar unzip zip"; do
  complete -F _fzf_all_completion -o default -o bashdefault $cmd
done

# Kill completion
complete -F _fzf_kill_completion -o nospace -o default -o bashdefault kill

# Host completion
complete -F _fzf_ssh_completion -o default -o bashdefault ssh
complete -F _fzf_telnet_completion -o default -o bashdefault telnet

