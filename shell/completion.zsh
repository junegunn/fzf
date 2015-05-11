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
  local base lbuf find_opts fzf_opts suffix tail fzf dir leftover matches nnm
  base=${(Q)1}
  lbuf=$2
  find_opts=$3
  fzf_opts=$4
  suffix=$5
  tail=$6
  [ ${FZF_TMUX:-1} -eq 1 ] && fzf="fzf-tmux -d ${FZF_TMUX_HEIGHT:-40%}" || fzf="fzf"

  if ! setopt | grep nonomatch > /dev/null; then
    nnm=1
    setopt nonomatch
  fi
  dir="$base"
  while [ 1 ]; do
    if [ -z "$dir" -o -d ${~dir} ]; then
      leftover=${base/#"$dir"}
      leftover=${leftover/#\/}
      [ "$dir" = './' ] && dir=''
      dir=${~dir}
      matches=$(\find -L $dir* ${=find_opts} 2> /dev/null | ${=fzf} ${=FZF_COMPLETION_OPTS} ${=fzf_opts} -q "$leftover" | while read item; do
        printf "%q$suffix " "$item"
      done)
      matches=${matches% }
      if [ -n "$matches" ]; then
        LBUFFER="$lbuf$matches$tail"
        zle redisplay
      fi
      break
    fi
    dir=$(dirname "$dir")
    dir=${dir%/}/
  done
  [ -n "$nnm" ] && unsetopt nonomatch
}

_fzf_all_completion() {
  _fzf_path_completion "$1" "$2" \
    "-name .git -prune -o -name .svn -prune -o -type d -print -o -type f -print -o -type l -print" \
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
    cat <(cat ~/.ssh/config /etc/ssh/ssh_config 2> /dev/null | \grep -i '^host' | \grep -v '*') <(\grep -v '^\s*\(#\|$\)' /etc/hosts | \grep -Fv '0.0.0.0') | awk '{if (length($2) > 0) {print $2}}' | sort -u
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

fzf-completion() {
  local tokens cmd prefix trigger tail fzf matches lbuf d_cmds

  # http://zsh.sourceforge.net/FAQ/zshfaq03.html
  # http://zsh.sourceforge.net/Doc/Release/Expansion.html#Parameter-Expansion-Flags
  tokens=(${(z)LBUFFER})
  if [ ${#tokens} -lt 1 ]; then
    eval "zle ${fzf_default_completion:-expand-or-complete}"
    return
  fi

  cmd=${tokens[1]}

  # Explicitly allow for empty trigger.
  trigger=${FZF_COMPLETION_TRIGGER-'**'}
  [ -z "$trigger" -a ${LBUFFER[-1]} = ' ' ] && tokens+=("")

  tail=${LBUFFER:$(( ${#LBUFFER} - ${#trigger} ))}
  # Kill completion (do not require trigger sequence)
  if [ $cmd = kill -a ${LBUFFER[-1]} = ' ' ]; then
    [ ${FZF_TMUX:-1} -eq 1 ] && fzf="fzf-tmux -d ${FZF_TMUX_HEIGHT:-40%}" || fzf="fzf"
    matches=$(ps -ef | sed 1d | ${=fzf} ${=FZF_COMPLETION_OPTS} -m | awk '{print $2}' | tr '\n' ' ')
    if [ -n "$matches" ]; then
      LBUFFER="$LBUFFER$matches"
      zle redisplay
    fi
  # Trigger sequence given
  elif [ ${#tokens} -gt 1 -a "$tail" = "$trigger" ]; then
    d_cmds=(cd pushd rmdir)

    [ -z "$trigger"      ] && prefix=${tokens[-1]} || prefix=${tokens[-1]:0:-${#trigger}}
    [ -z "${tokens[-1]}" ] && lbuf=$LBUFFER        || lbuf=${LBUFFER:0:-${#tokens[-1]}}

    if [ ${d_cmds[(i)$cmd]} -le ${#d_cmds} ]; then
      _fzf_dir_completion "$prefix" $lbuf
    elif [ $cmd = telnet ]; then
      _fzf_telnet_completion "$prefix" $lbuf
    elif [ $cmd = ssh ]; then
      _fzf_ssh_completion "$prefix" $lbuf
    elif [ $cmd = unset -o $cmd = export ]; then
      _fzf_env_var_completion "$prefix" $lbuf
    elif [ $cmd = unalias ]; then
      _fzf_alias_completion "$prefix" $lbuf
    else
      _fzf_all_completion "$prefix" $lbuf
    fi
  # Fall back to default completion
  else
    eval "zle ${fzf_default_completion:-expand-or-complete}"
  fi
}

[ -z "$fzf_default_completion" ] &&
  fzf_default_completion=$(bindkey '^I' | grep -v undefined-key | awk '{print $2}')

zle     -N   fzf-completion
bindkey '^I' fzf-completion

