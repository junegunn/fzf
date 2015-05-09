#!/bin/zsh
#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/-completion.zsh
#
# - $FZF_TMUX               (default: 1)
# - $FZF_TMUX_HEIGHT        (default: '40%')
# - $FZF_COMPLETION_TRIGGER (default: '**')
# - $FZF_COMPLETION_OPTS    (default: empty)

_fzf_path_completion() {
  local base lbuf find_opts fzf_opts suffix tail fzf dir leftover matches
  base=$1
  lbuf=$2
  find_opts=$3
  fzf_opts=$4
  suffix=$5
  tail=$6
  [ ${FZF_TMUX:-1} -eq 1 ] && fzf="fzf-tmux -d ${FZF_TMUX_HEIGHT:-40%}" || fzf="fzf"

  dir="$base"
  while [ 1 ]; do
    if [ -z "$dir" -o -d ${~dir} ]; then
      leftover=${base/#"$dir"}
      leftover=${leftover/#\/}
      [ "$dir" = './' ] && dir=''
      matches=$(\find -L ${~dir}* ${=find_opts} 2> /dev/null | ${=fzf} ${=FZF_COMPLETION_OPTS} ${=fzf_opts} -q "$leftover" | while read item; do
        printf "%q$suffix " "$item"
      done)
      matches=${matches% }
      if [ -n "$matches" ]; then
        LBUFFER="$lbuf$matches$tail"
        zle redisplay
      fi
      return
    fi
    dir=$(dirname "$dir")
    [[ "$dir" =~ /$ ]] || dir="$dir"/
  done
}

_fzf_all_completion() {
  _fzf_path_completion "$1" "$2" \
    "-name .git -prune -o -name .svn -prune -o -type d -print -o -type f -print -o -type l -print" \
    "-m" "" " "
}

_fzf_file_completion() {
  _fzf_path_completion "$1" "$2" \
    "-name .git -prune -o -name .svn -prune -o -type f -print -o -type l -print" \
    "-m" "" " "
}

_fzf_dir_completion() {
  _fzf_path_completion "$1" "$2" \
    "-name .git -prune -o -name .svn -prune -o -type d -print" \
    "" "/" ""
}

_fzf_list_completion() {
  local prefix lbuf fzf_opts src fzf matches
  prefix=$1
  lbuf=$2
  fzf_opts=$3
  read -r src
  [ ${FZF_TMUX:-1} -eq 1 ] && fzf="fzf-tmux -d ${FZF_TMUX_HEIGHT:-40%}" || fzf="fzf"

  matches=$(eval "$src" | ${=fzf} ${=FZF_COMPLETION_OPTS} ${=fzf_opts} -q "$prefix")
  if [ -n "$matches" ]; then
    LBUFFER="$lbuf$matches "
    zle redisplay
  fi
}

_fzf_telnet_completion() {
  _fzf_list_completion "$1" "$2" '+m' << "EOF"
  \grep -v '^\s*\(#\|$\)' /etc/hosts | \grep -Fv '0.0.0.0' | awk '{if (length($2) > 0) {print $2}}' | sort -u
EOF
}

_fzf_ssh_completion() {
  _fzf_list_completion "$1" "$2" '+m' << "EOF"
    cat <(cat ~/.ssh/config /etc/ssh/ssh_config 2> /dev/null | \grep -i ^host | \grep -v '*') <(\grep -v '^\s*\(#\|$\)' /etc/hosts | \grep -Fv '0.0.0.0') | awk '{if (length($2) > 0) {print $2}}' | sort -u
EOF
}

_fzf_env_var_completion() {
  _fzf_list_completion "$1" "$2" '+m' << "EOF"
  declare -xp | sed 's/=.*//' | sed 's/.* //'
EOF
}

_fzf_alias_completion() {
  _fzf_list_completion "$1" "$2" '+m' << "EOF"
  alias | sed 's/=.*//'
EOF
}

fzf-zsh-completion() {
  local tokens cmd prefix trigger tail fzf matches lbuf d_cmds f_cmds a_cmds

  # http://zsh.sourceforge.net/FAQ/zshfaq03.html
  tokens=(${=LBUFFER})
  if [ ${#tokens} -lt 1 ]; then
    zle expand-or-complete
    return
  fi

  cmd=${tokens[1]}
  trigger=${FZF_COMPLETION_TRIGGER:-**}

  # Trigger sequence given
  tail=${LBUFFER:$(( ${#LBUFFER} - ${#trigger} ))}
  if [ ${#tokens} -gt 1 -a $tail = $trigger ]; then
    d_cmds=(cd pushd rmdir)
    f_cmds=(
      awk cat diff diff3
      emacs ex file ftp g++ gcc gvim head hg java
      javac ld less more mvim patch perl python ruby
      sed sftp sort source tail tee uniq vi view vim wc)
    a_cmds=(
      basename bunzip2 bzip2 chmod chown curl cp dirname du
      find git grep gunzip gzip hg jar
      ln ls mv open rm rsync scp
      svn tar unzip zip)

    prefix=${tokens[-1]:0:-${#trigger}}
    lbuf=${LBUFFER:0:-${#tokens[-1]}}
    if [ ${d_cmds[(i)$cmd]} -le ${#d_cmds} ]; then
      _fzf_dir_completion "$prefix" $lbuf
    elif [ ${f_cmds[(i)$cmd]} -le ${#f_cmds} ]; then
      _fzf_file_completion "$prefix" $lbuf
    elif [ ${a_cmds[(i)$cmd]} -le ${#a_cmds} ]; then
      _fzf_all_completion "$prefix" $lbuf
    elif [ $cmd = telnet ]; then
      _fzf_telnet_completion "$prefix" $lbuf
    elif [ $cmd = ssh ]; then
      _fzf_ssh_completion "$prefix" $lbuf
    elif [ $cmd = unset -o $cmd = export ]; then
      _fzf_env_var_completion "$prefix" $lbuf
    elif [ $cmd = unalias ]; then
      _fzf_alias_completion "$prefix" $lbuf
    fi
  # Kill completion (do not require trigger sequence)
  elif [ $cmd = kill -a ${LBUFFER[-1]} = ' ' ]; then
    [ ${FZF_TMUX:-1} -eq 1 ] && fzf="fzf-tmux -d ${FZF_TMUX_HEIGHT:-40%}" || fzf="fzf"
    matches=$(ps -ef | sed 1d | ${=fzf} ${=FZF_COMPLETION_OPTS} -m | awk '{print $2}' | tr '\n' ' ')
    if [ -n "$matches" ]; then
      LBUFFER="$LBUFFER$matches"
      zle redisplay
    fi
  # Fall back to default completion
  else
    zle expand-or-complete
  fi
}

zle     -N   fzf-zsh-completion
bindkey '^I' fzf-zsh-completion

