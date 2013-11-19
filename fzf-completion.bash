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
  local cur prev opts base matches ignore
  COMPREPLY=()
  FZF_COMPLETION_TRIGGER=${FZF_COMPLETION_TRIGGER:-**}
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  if [[ ${cur} == *"$FZF_COMPLETION_TRIGGER" ]]; then
    base=${cur:0:${#cur}-${#FZF_COMPLETION_TRIGGER}}
    base=${base%/}
    eval base=$base

    ignore=${FZF_COMPLETION_IGNORE:-*/.git/*}
    find_opts="-name .git -prune -o -name .svn -prune -o"
    if [ -z "$base" -o -d "$base" ]; then
      matches=$(find ${base:-*} $1 2> /dev/null | fzf $FZF_COMPLETION_OPTS $2 | while read item; do
        if [[ ${item} =~ \  ]]; then
          echo -n "\"$item\" "
        else
          echo -n "$item "
        fi
      done)
      matches=${matches% }
      if [ -n "$matches" ]; then
        COMPREPLY=( "$matches" )
        return 0
      fi
    fi
  fi
}

_fzf_all_completion() {
  _fzf_generic_completion \
    "-name .git -prune -o -name .svn -prune -o -type f -print -o -type l -print" \
    "-m"
}

_fzf_dir_completion() {
  _fzf_generic_completion \
    "-name .git -prune -o -name .svn -prune -o -type d -print" \
    ""
}

complete -F _fzf_opts_completion fzf

for cmd in "cd pushd rmdir"; do
  complete -F _fzf_dir_completion -o default $cmd
done

FZF_COMPLETION_COMMANDS=${FZF_COMPLETION_COMMANDS:-
  awk basename bunzip2 bzip2 curl diff diff3 dirname du emacs ex file find ftp
  g++ gcc git grep gunzip gvim gzip head hg jar java javac jps ld less ls more
  mvim open patch perl python rsync ruby scp sed sftp sort svn tail tar tee
  uniq unzip vi view vim wc zip
}

for cmd in $FZF_COMPLETION_COMMANDS; do
  complete -F _fzf_all_completion -o default $cmd
done

bind '"\e\e": complete'
bind '"\er": redraw-current-line'
bind '"\C-i": "\e\e\er"'

