#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ completion.zsh
#
# - $FZF_TMUX               (default: 0)
# - $FZF_TMUX_OPTS          (default: '-d 40%')
# - $FZF_COMPLETION_TRIGGER (default: '**')
# - $FZF_COMPLETION_OPTS    (default: empty)

# Both branches of the following `if` do the same thing -- define
# __fzf_completion_options such that `eval $__fzf_completion_options` sets
# all options to the same values they currently have. We'll do just that at
# the bottom of the file after changing options to what we prefer.
#
# IMPORTANT: Until we get to the `emulate` line, all words that *can* be quoted
# *must* be quoted in order to prevent alias expansion. In addition, code must
# be written in a way works with any set of zsh options. This is very tricky, so
# careful when you change it.
#
# Start by loading the builtin zsh/parameter module. It provides `options`
# associative array that stores current shell options.
if 'zmodload' 'zsh/parameter' 2>'/dev/null' && (( ${+options} )); then
  # This is the fast branch and it gets taken on virtually all Zsh installations.
  #
  # ${(kv)options[@]} expands to array of keys (option names) and values ("on"
  # or "off"). The subsequent expansion# with (j: :) flag joins all elements
  # together separated by spaces. __fzf_completion_options ends up with a value
  # like this: "options=(shwordsplit off aliases on ...)".
  __fzf_completion_options="options=(${(j: :)${(kv)options[@]}})"
else
  # This branch is much slower because it forks to get the names of all
  # zsh options. It's possible to eliminate this fork but it's not worth the
  # trouble because this branch gets taken only on very ancient or broken
  # zsh installations.
  () {
    # That `()` above defines an anonymous function. This is essentially a scope
    # for local parameters. We use it to avoid polluting global scope.
    'local' '__fzf_opt'
    __fzf_completion_options="setopt"
    # `set -o` prints one line for every zsh option. Each line contains option
    # name, some spaces, and then either "on" or "off". We just want option names.
    # Expansion with (@f) flag splits a string into lines. The outer expansion
    # removes spaces and everything that follow them on every line. __fzf_opt
    # ends up iterating over option names: shwordsplit, aliases, etc.
    for __fzf_opt in "${(@)${(@f)$(set -o)}%% *}"; do
      if [[ -o "$__fzf_opt" ]]; then
        # Option $__fzf_opt is currently on, so remember to set it back on.
        __fzf_completion_options+=" -o $__fzf_opt"
      else
        # Option $__fzf_opt is currently off, so remember to set it back off.
        __fzf_completion_options+=" +o $__fzf_opt"
      fi
    done
    # The value of __fzf_completion_options here looks like this:
    # "setopt +o shwordsplit -o aliases ..."
  }
fi

# Enable the default zsh options (those marked with <Z> in `man zshoptions`)
# but without `aliases`. Aliases in functions are expanded when functions are
# defined, so if we disable aliases here, we'll be sure to have no pesky
# aliases in any of our functions. This way we won't need prefix every
# command with `command` or to quote every word to defend against global
# aliases. Note that `aliases` is not the only option that's important to
# control. There are several others that could wreck havoc if they are set
# to values we don't expect. With the following `emulate` command we
# sidestep this issue entirely.
'emulate' 'zsh' '-o' 'no_aliases'

# This brace is the start of try-always block. The `always` part is like
# `finally` in lesser languages. We use it to *always* restore user options.
{

# Bail out if not interactive shell.
[[ -o interactive ]] || return 0

# To use custom commands instead of find, override _fzf_compgen_{path,dir}
if ! declare -f _fzf_compgen_path > /dev/null; then
  _fzf_compgen_path() {
    echo "$1"
    command find -L "$1" \
      -name .git -prune -o -name .hg -prune -o -name .svn -prune -o \( -type d -o -type f -o -type l \) \
      -a -not -path "$1" -print 2> /dev/null | sed 's@^\./@@'
  }
fi

if ! declare -f _fzf_compgen_dir > /dev/null; then
  _fzf_compgen_dir() {
    command find -L "$1" \
      -name .git -prune -o -name .hg -prune -o -name .svn -prune -o -type d \
      -a -not -path "$1" -print 2> /dev/null | sed 's@^\./@@'
  }
fi

###########################################################

__fzf_comprun() {
  if [[ "$(type _fzf_comprun 2>&1)" =~ function ]]; then
    _fzf_comprun "$@"
  elif [ -n "$TMUX_PANE" ] && { [ "${FZF_TMUX:-0}" != 0 ] || [ -n "$FZF_TMUX_OPTS" ]; }; then
    shift
    if [ -n "$FZF_TMUX_OPTS" ]; then
      fzf-tmux ${(Q)${(Z+n+)FZF_TMUX_OPTS}} -- "$@"
    else
      fzf-tmux -d ${FZF_TMUX_HEIGHT:-40%} -- "$@"
    fi
  else
    shift
    fzf "$@"
  fi
}

# Extract the name of the command. e.g. foo=1 bar baz**<tab>
__fzf_extract_command() {
  local token tokens
  tokens=(${(z)1})
  for token in $tokens; do
    token=${(Q)token}
    if [[ "$token" =~ [[:alnum:]] && ! "$token" =~ "=" ]]; then
      echo "$token"
      return
    fi
  done
  echo "${tokens[1]}"
}

__fzf_generic_path_completion() {
  local base lbuf cmd compgen fzf_opts suffix tail dir leftover matches
  base=$1
  lbuf=$2
  cmd=$(__fzf_extract_command "$lbuf")
  compgen=$3
  fzf_opts=$4
  suffix=$5
  tail=$6

  setopt localoptions nonomatch
  eval "base=$base"
  [[ $base = *"/"* ]] && dir="$base"
  while [ 1 ]; do
    if [[ -z "$dir" || -d ${dir} ]]; then
      leftover=${base/#"$dir"}
      leftover=${leftover/#\/}
      [ -z "$dir" ] && dir='.'
      [ "$dir" != "/" ] && dir="${dir/%\//}"
      matches=$(eval "$compgen $(printf %q "$dir")" | FZF_DEFAULT_OPTS="--height ${FZF_TMUX_HEIGHT:-40%} --reverse $FZF_DEFAULT_OPTS $FZF_COMPLETION_OPTS" __fzf_comprun "$cmd" ${(Q)${(Z+n+)fzf_opts}} -q "$leftover" | while read item; do
        echo -n "${(q)item}$suffix "
      done)
      matches=${matches% }
      if [ -n "$matches" ]; then
        LBUFFER="$lbuf$matches$tail"
      fi
      zle reset-prompt
      break
    fi
    dir=$(dirname "$dir")
    dir=${dir%/}/
  done
}

_fzf_path_completion() {
  __fzf_generic_path_completion "$1" "$2" _fzf_compgen_path \
    "-m" "" " "
}

_fzf_dir_completion() {
  __fzf_generic_path_completion "$1" "$2" _fzf_compgen_dir \
    "" "/" ""
}

_fzf_feed_fifo() (
  command rm -f "$1"
  mkfifo "$1"
  cat <&0 > "$1" &
)

_fzf_complete() {
  setopt localoptions ksh_arrays
  # Split arguments around --
  local args rest str_arg i sep
  args=("$@")
  sep=
  for i in {0..${#args[@]}}; do
    if [[ "${args[$i]}" = -- ]]; then
      sep=$i
      break
    fi
  done
  if [[ -n "$sep" ]]; then
    str_arg=
    rest=("${args[@]:$((sep + 1)):${#args[@]}}")
    args=("${args[@]:0:$sep}")
  else
    str_arg=$1
    args=()
    shift
    rest=("$@")
  fi

  local fifo lbuf cmd matches post
  fifo="${TMPDIR:-/tmp}/fzf-complete-fifo-$$"
  lbuf=${rest[0]}
  cmd=$(__fzf_extract_command "$lbuf")
  post="${funcstack[1]}_post"
  type $post > /dev/null 2>&1 || post=cat

  _fzf_feed_fifo "$fifo"
  matches=$(FZF_DEFAULT_OPTS="--height ${FZF_TMUX_HEIGHT:-40%} --reverse $FZF_DEFAULT_OPTS $FZF_COMPLETION_OPTS $str_arg" __fzf_comprun "$cmd" "${args[@]}" -q "${(Q)prefix}" < "$fifo" | $post | tr '\n' ' ')
  if [ -n "$matches" ]; then
    LBUFFER="$lbuf$matches"
  fi
  zle reset-prompt
  command rm -f "$fifo"
}

_fzf_complete_telnet() {
  _fzf_complete +m -- "$@" < <(
    command grep -v '^\s*\(#\|$\)' /etc/hosts | command grep -Fv '0.0.0.0' |
        awk '{if (length($2) > 0) {print $2}}' | sort -u
  )
}

_fzf_complete_ssh() {
  _fzf_complete +m -- "$@" < <(
    setopt localoptions nonomatch
    command cat <(cat ~/.ssh/config ~/.ssh/config.d/* /etc/ssh/ssh_config 2> /dev/null | command grep -i '^\s*host\(name\)\? ' | awk '{for (i = 2; i <= NF; i++) print $1 " " $i}' | command grep -v '[*?]') \
        <(command grep -oE '^[[a-z0-9.,:-]+' ~/.ssh/known_hosts | tr ',' '\n' | tr -d '[' | awk '{ print $1 " " $1 }') \
        <(command grep -v '^\s*\(#\|$\)' /etc/hosts | command grep -Fv '0.0.0.0') |
        awk '{if (length($2) > 0) {print $2}}' | sort -u
  )
}

_fzf_complete_export() {
  _fzf_complete -m -- "$@" < <(
    declare -xp | sed 's/=.*//' | sed 's/.* //'
  )
}

_fzf_complete_unset() {
  _fzf_complete -m -- "$@" < <(
    declare -xp | sed 's/=.*//' | sed 's/.* //'
  )
}

_fzf_complete_unalias() {
  _fzf_complete +m -- "$@" < <(
    alias | sed 's/=.*//'
  )
}

_fzf_complete_kill() {
  _fzf_complete -m --preview 'echo {}' --preview-window down:3:wrap --min-height 15 -- "$@" < <(
    command ps -ef | sed 1d
  )
}

_fzf_complete_kill_post() {
  awk '{print $2}'
}

fzf-completion() {
  local tokens cmd prefix trigger tail matches lbuf d_cmds
  setopt localoptions noshwordsplit noksh_arrays noposixbuiltins

  # http://zsh.sourceforge.net/FAQ/zshfaq03.html
  # http://zsh.sourceforge.net/Doc/Release/Expansion.html#Parameter-Expansion-Flags
  tokens=(${(z)LBUFFER})
  if [ ${#tokens} -lt 1 ]; then
    zle ${fzf_default_completion:-expand-or-complete}
    return
  fi

  cmd=$(__fzf_extract_command "$LBUFFER")

  # Explicitly allow for empty trigger.
  trigger=${FZF_COMPLETION_TRIGGER-'**'}
  [ -z "$trigger" -a ${LBUFFER[-1]} = ' ' ] && tokens+=("")

  # When the trigger starts with ';', it becomes a separate token
  if [[ ${LBUFFER} = *"${tokens[-2]}${tokens[-1]}" ]]; then
    tokens[-2]="${tokens[-2]}${tokens[-1]}"
    tokens=(${tokens[0,-2]})
  fi

  lbuf=$LBUFFER
  tail=${LBUFFER:$(( ${#LBUFFER} - ${#trigger} ))}
  # Kill completion (do not require trigger sequence)
  if [ "$cmd" = kill -a ${LBUFFER[-1]} = ' ' ]; then
    tail=$trigger
    tokens+=$trigger
    lbuf="$lbuf$trigger"
  fi

  # Trigger sequence given
  if [ ${#tokens} -gt 1 -a "$tail" = "$trigger" ]; then
    d_cmds=(${=FZF_COMPLETION_DIR_COMMANDS:-cd pushd rmdir})

    [ -z "$trigger"      ] && prefix=${tokens[-1]} || prefix=${tokens[-1]:0:-${#trigger}}
    [ -n "${tokens[-1]}" ] && lbuf=${lbuf:0:-${#tokens[-1]}}

    if eval "type _fzf_complete_${cmd} > /dev/null"; then
      prefix="$prefix" eval _fzf_complete_${cmd} ${(q)lbuf}
    elif [ ${d_cmds[(i)$cmd]} -le ${#d_cmds} ]; then
      _fzf_dir_completion "$prefix" "$lbuf"
    else
      _fzf_path_completion "$prefix" "$lbuf"
    fi
  # Fall back to default completion
  else
    zle ${fzf_default_completion:-expand-or-complete}
  fi
}

[ -z "$fzf_default_completion" ] && {
  binding=$(bindkey '^I')
  [[ $binding =~ 'undefined-key' ]] || fzf_default_completion=$binding[(s: :w)2]
  unset binding
}

zle     -N   fzf-completion
bindkey '^I' fzf-completion

} always {
  # Restore the original options.
  eval $__fzf_completion_options
  'unset' '__fzf_completion_options'
}
