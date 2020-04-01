#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ completion.bash
#
# - $FZF_TMUX               (default: 0)
# - $FZF_TMUX_OPTS          (default: empty)
# - $FZF_COMPLETION_TRIGGER (default: '**')
# - $FZF_COMPLETION_OPTS    (default: empty)

if [[ $- =~ i ]]; then

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

# To redraw line after fzf closes (printf '\e[5n')
bind '"\e[0n": redraw-current-line'

__fzf_comprun() {
  if [ "$(type -t _fzf_comprun 2>&1)" = function ]; then
    _fzf_comprun "$@"
  elif [ -n "$TMUX_PANE" ] && { [ "${FZF_TMUX:-0}" != 0 ] || [ -n "$FZF_TMUX_OPTS" ]; }; then
    shift
    fzf-tmux ${FZF_TMUX_OPTS:--d${FZF_TMUX_HEIGHT:-40%}} -- "$@"
  else
    shift
    fzf "$@"
  fi
}

__fzf_orig_completion_filter() {
  sed 's/^\(.*-F\) *\([^ ]*\).* \([^ ]*\)$/export _fzf_orig_completion_\3="\1 %s \3 #\2"; [[ "\1" = *" -o nospace "* ]] \&\& [[ ! "$__fzf_nospace_commands" = *" \3 "* ]] \&\& __fzf_nospace_commands="$__fzf_nospace_commands \3 ";/' |
  awk -F= '{OFS = FS} {gsub(/[^A-Za-z0-9_= ;]/, "_", $1);}1'
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
    --pointer
    --marker
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

  return 0
}

_fzf_handle_dynamic_completion() {
  local cmd orig_var orig ret orig_cmd orig_complete
  cmd="$1"
  shift
  orig_cmd="$1"
  orig_var="_fzf_orig_completion_$cmd"
  orig="${!orig_var##*#}"
  if [ -n "$orig" ] && type "$orig" > /dev/null 2>&1; then
    $orig "$@"
  elif [ -n "$_fzf_completion_loader" ]; then
    orig_complete=$(complete -p "$orig_cmd" 2> /dev/null)
    _completion_loader "$@"
    ret=$?
    # _completion_loader may not have updated completion for the command
    if [ "$(complete -p "$orig_cmd" 2> /dev/null)" != "$orig_complete" ]; then
      eval "$(complete | command grep " -F.* $orig_cmd$" | __fzf_orig_completion_filter)"
      if [[ "$__fzf_nospace_commands" = *" $orig_cmd "* ]]; then
        eval "${orig_complete/ -F / -o nospace -F }"
      else
        eval "$orig_complete"
      fi
    fi
    return $ret
  fi
}

__fzf_generic_path_completion() {
  local cur base dir leftover matches trigger cmd
  cmd="${COMP_WORDS[0]//[^A-Za-z0-9_=]/_}"
  COMPREPLY=()
  trigger=${FZF_COMPLETION_TRIGGER-'**'}
  cur="${COMP_WORDS[COMP_CWORD]}"
  if [[ "$cur" == *"$trigger" ]]; then
    base=${cur:0:${#cur}-${#trigger}}
    eval "base=$base"

    [[ $base = *"/"* ]] && dir="$base"
    while true; do
      if [ -z "$dir" ] || [ -d "$dir" ]; then
        leftover=${base/#"$dir"}
        leftover=${leftover/#\/}
        [ -z "$dir" ] && dir='.'
        [ "$dir" != "/" ] && dir="${dir/%\//}"
        matches=$(eval "$1 $(printf %q "$dir")" | FZF_DEFAULT_OPTS="--height ${FZF_TMUX_HEIGHT:-40%} --reverse $FZF_DEFAULT_OPTS $FZF_COMPLETION_OPTS $2" __fzf_comprun "$4" -q "$leftover" | while read -r item; do
          printf "%q$3 " "$item"
        done)
        matches=${matches% }
        [[ -z "$3" ]] && [[ "$__fzf_nospace_commands" = *" ${COMP_WORDS[0]} "* ]] && matches="$matches "
        if [ -n "$matches" ]; then
          COMPREPLY=( "$matches" )
        else
          COMPREPLY=( "$cur" )
        fi
        printf '\e[5n'
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
  # Split arguments around --
  local args rest str_arg i sep
  args=("$@")
  sep=
  for i in "${!args[@]}"; do
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

  local cur selected trigger cmd post
  post="$(caller 0 | awk '{print $2}')_post"
  type -t "$post" > /dev/null 2>&1 || post=cat

  cmd="${COMP_WORDS[0]//[^A-Za-z0-9_=]/_}"
  trigger=${FZF_COMPLETION_TRIGGER-'**'}
  cur="${COMP_WORDS[COMP_CWORD]}"
  if [[ "$cur" == *"$trigger" ]]; then
    cur=${cur:0:${#cur}-${#trigger}}

    selected=$(FZF_DEFAULT_OPTS="--height ${FZF_TMUX_HEIGHT:-40%} --reverse $FZF_DEFAULT_OPTS $FZF_COMPLETION_OPTS $str_arg" __fzf_comprun "${rest[0]}" "${args[@]}" -q "$cur" | $post | tr '\n' ' ')
    selected=${selected% } # Strip trailing space not to repeat "-o nospace"
    if [ -n "$selected" ]; then
      COMPREPLY=("$selected")
    else
      COMPREPLY=("$cur")
    fi
    printf '\e[5n'
    return 0
  else
    _fzf_handle_dynamic_completion "$cmd" "${rest[@]}"
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

  local selected
  selected=$(command ps -ef | sed 1d | FZF_DEFAULT_OPTS="--height ${FZF_TMUX_HEIGHT:-50%} --min-height 15 --reverse $FZF_DEFAULT_OPTS $FZF_COMPLETION_OPTS --preview 'echo {}' --preview-window down:3:wrap" __fzf_comprun "kill" -m | awk '{print $2}' | tr '\n' ' ')
  selected=${selected% }
  printf '\e[5n'

  if [ -n "$selected" ]; then
    COMPREPLY=( "$selected" )
    return 0
  fi
}

_fzf_host_completion() {
  _fzf_complete +m -- "$@" < <(
    cat <(cat ~/.ssh/config ~/.ssh/config.d/* /etc/ssh/ssh_config 2> /dev/null | command grep -i '^\s*host\(name\)\? ' | awk '{for (i = 2; i <= NF; i++) print $1 " " $i}' | command grep -v '[*?]') \
        <(command grep -oE '^[[a-z0-9.,:-]+' ~/.ssh/known_hosts | tr ',' '\n' | tr -d '[' | awk '{ print $1 " " $1 }') \
        <(command grep -v '^\s*\(#\|$\)' /etc/hosts | command grep -Fv '0.0.0.0') |
        awk '{if (length($2) > 0) {print $2}}' | sort -u
  )
}

_fzf_var_completion() {
  _fzf_complete -m -- "$@" < <(
    declare -xp | sed 's/=.*//' | sed 's/.* //'
  )
}

_fzf_alias_completion() {
  _fzf_complete -m -- "$@" < <(
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
x_cmds="kill"

# Preserve existing completion
eval "$(complete |
  sed -E '/-F/!d; / _fzf/d; '"/ ($(echo $d_cmds $a_cmds $x_cmds | sed 's/ /|/g; s/+/\\+/g'))$/"'!d' |
  __fzf_orig_completion_filter)"

if type _completion_loader > /dev/null 2>&1; then
  _fzf_completion_loader=1
fi

__fzf_defc() {
  local cmd func opts orig_var orig def
  cmd="$1"
  func="$2"
  opts="$3"
  orig_var="_fzf_orig_completion_${cmd//[^A-Za-z0-9_]/_}"
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
  __fzf_defc "$cmd" _fzf_path_completion "-o default -o bashdefault"
done

# Directory
for cmd in $d_cmds; do
  __fzf_defc "$cmd" _fzf_dir_completion "-o nospace -o dirnames"
done

# Kill completion
complete -F _fzf_complete_kill -o default -o bashdefault kill

unset cmd d_cmds a_cmds x_cmds

_fzf_setup_completion() {
  local kind fn cmd
  kind=$1
  fn=_fzf_${1}_completion
  if [[ $# -lt 2 ]] || ! type -t "$fn" > /dev/null; then
    echo "usage: ${FUNCNAME[0]} path|dir|var|alias|host COMMANDS..."
    return 1
  fi
  shift
  eval "$(complete -p "$@" 2> /dev/null | grep -v "$fn" | __fzf_orig_completion_filter)"
  for cmd in "$@"; do
    case "$kind" in
      dir)   __fzf_defc "$cmd" "$fn" "-o nospace -o dirnames" ;;
      var)   __fzf_defc "$cmd" "$fn" "-o default -o nospace -v" ;;
      alias) __fzf_defc "$cmd" "$fn" "-a" ;;
      *)     __fzf_defc "$cmd" "$fn" "-o default -o bashdefault" ;;
    esac
  done
}

# Environment variables / Aliases / Hosts
_fzf_setup_completion 'var'   export unset
_fzf_setup_completion 'alias' unalias
_fzf_setup_completion 'host'  ssh telnet

fi
