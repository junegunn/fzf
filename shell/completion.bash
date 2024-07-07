#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ completion.bash
#
# - $FZF_TMUX                 (default: 0)
# - $FZF_TMUX_OPTS            (default: empty)
# - $FZF_COMPLETION_TRIGGER   (default: '**')
# - $FZF_COMPLETION_OPTS      (default: empty)
# - $FZF_COMPLETION_PATH_OPTS (default: empty)
# - $FZF_COMPLETION_DIR_OPTS  (default: empty)

if [[ $- =~ i ]]; then


# To use custom commands instead of find, override _fzf_compgen_{path,dir}
#
#   _fzf_compgen_path() {
#     echo "$1"
#     command find -L "$1" \
#       -name .git -prune -o -name .hg -prune -o -name .svn -prune -o \( -type d -o -type f -o -type l \) \
#       -a -not -path "$1" -print 2> /dev/null | command sed 's@^\./@@'
#   }
#
#   _fzf_compgen_dir() {
#     command find -L "$1" \
#       -name .git -prune -o -name .hg -prune -o -name .svn -prune -o -type d \
#       -a -not -path "$1" -print 2> /dev/null | command sed 's@^\./@@'
#   }

###########################################################

# To redraw line after fzf closes (printf '\e[5n')
bind '"\e[0n": redraw-current-line' 2> /dev/null

__fzf_defaults() {
  # $1: Prepend to FZF_DEFAULT_OPTS_FILE and FZF_DEFAULT_OPTS
  # $2: Append to FZF_DEFAULT_OPTS_FILE and FZF_DEFAULT_OPTS
  echo "--height ${FZF_TMUX_HEIGHT:-40%} --bind=ctrl-z:ignore $1"
  command cat "${FZF_DEFAULT_OPTS_FILE-}" 2> /dev/null
  echo "${FZF_DEFAULT_OPTS-} $2"
}

__fzf_comprun() {
  if [[ "$(type -t _fzf_comprun 2>&1)" = function ]]; then
    _fzf_comprun "$@"
  elif [[ -n "${TMUX_PANE-}" ]] && { [[ "${FZF_TMUX:-0}" != 0 ]] || [[ -n "${FZF_TMUX_OPTS-}" ]]; }; then
    shift
    fzf-tmux ${FZF_TMUX_OPTS:--d${FZF_TMUX_HEIGHT:-40%}} -- "$@"
  else
    shift
    fzf "$@"
  fi
}

__fzf_orig_completion() {
  local l comp f cmd
  while read -r l; do
    if [[ "$l" =~ ^(.*\ -F)\ *([^ ]*).*\ ([^ ]*)$ ]]; then
      comp="${BASH_REMATCH[1]}"
      f="${BASH_REMATCH[2]}"
      cmd="${BASH_REMATCH[3]}"
      [[ "$f" = _fzf_* ]] && continue
      printf -v "_fzf_orig_completion_${cmd//[^A-Za-z0-9_]/_}" "%s" "${comp} %s ${cmd} #${f}"
      if [[ "$l" = *" -o nospace "* ]] && [[ ! "${__fzf_nospace_commands-}" = *" $cmd "* ]]; then
        __fzf_nospace_commands="${__fzf_nospace_commands-} $cmd "
      fi
    fi
  done
}

# @param $1 cmd - Command name for which the original completion is searched
# @var[out] REPLY - Original function name is returned
__fzf_orig_completion_get_orig_func() {
  local cmd orig_var orig
  cmd=$1
  orig_var="_fzf_orig_completion_${cmd//[^A-Za-z0-9_]/_}"
  orig="${!orig_var-}"
  REPLY="${orig##*#}"
  [[ $REPLY ]] && type "$REPLY" &> /dev/null
}

# @param $1 cmd - Command name for which the original completion is searched
# @param $2 func - Fzf's completion function to replace the original function
# @var[out] REPLY - Completion setting is returned as a string to "eval"
__fzf_orig_completion_instantiate() {
  local cmd func orig_var orig
  cmd=$1
  func=$2
  orig_var="_fzf_orig_completion_${cmd//[^A-Za-z0-9_]/_}"
  orig="${!orig_var-}"
  orig="${orig%#*}"
  [[ $orig == *' %s '* ]] || return 1
  printf -v REPLY "$orig" "$func"
}

_fzf_opts_completion() {
  local cur prev opts
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  opts="
    +c --no-color
    +i --no-ignore-case
    +s --no-sort
    +x --no-extended
    --ansi
    --bash
    --bind
    --border
    --border-label
    --border-label-pos
    --color
    --cycle
    --disabled
    --ellipsis
    --expect
    --filepath-word
    --fish
    --header
    --header-first
    --header-lines
    --height
    --highlight-line
    --history
    --history-size
    --hscroll-off
    --info
    --jump-labels
    --keep-right
    --layout
    --listen
    --listen-unsafe
    --literal
    --man
    --margin
    --marker
    --min-height
    --no-bold
    --no-clear
    --no-hscroll
    --no-mouse
    --no-scrollbar
    --no-separator
    --no-unicode
    --padding
    --pointer
    --preview
    --preview-label
    --preview-label-pos
    --preview-window
    --print-query
    --print0
    --prompt
    --read0
    --reverse
    --scheme
    --scroll-off
    --separator
    --sync
    --tabstop
    --tac
    --tiebreak
    --tmux
    --track
    --version
    --with-nth
    --with-shell
    --wrap
    --zsh
    -0 --exit-0
    -1 --select-1
    -d --delimiter
    -e --exact
    -f --filter
    -h --help
    -i --ignore-case
    -m --multi
    -n --nth
    -q --query
    --"

  case "${prev}" in
  --scheme)
    COMPREPLY=( $(compgen -W "default path history" -- "$cur") )
    return 0
    ;;
  --tiebreak)
    COMPREPLY=( $(compgen -W "length chunk begin end index" -- "$cur") )
    return 0
    ;;
  --color)
    COMPREPLY=( $(compgen -W "dark light 16 bw no" -- "$cur") )
    return 0
    ;;
  --layout)
    COMPREPLY=( $(compgen -W "default reverse reverse-list" -- "$cur") )
    return 0
    ;;
  --info)
    COMPREPLY=( $(compgen -W "default right hidden inline inline-right" -- "$cur") )
    return 0
    ;;
  --preview-window)
    COMPREPLY=( $(compgen -W "
      default
      hidden
      nohidden
      wrap
      nowrap
      cycle
      nocycle
      up top
      down bottom
      left
      right
      rounded border border-rounded
      sharp border-sharp
      border-bold
      border-block
      border-thinblock
      border-double
      noborder border-none
      border-horizontal
      border-vertical
      border-up border-top
      border-down border-bottom
      border-left
      border-right
      follow
      nofollow" -- "$cur") )
    return 0
    ;;
  --border)
    COMPREPLY=( $(compgen -W "rounded sharp bold block thinblock double horizontal vertical top bottom left right none" -- "$cur") )
    return 0
    ;;
  --border-label-pos|--preview-label-pos)
    COMPREPLY=( $(compgen -W "center bottom top" -- "$cur") )
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
  local cmd ret REPLY orig_cmd orig_complete
  cmd="$1"
  shift
  orig_cmd="$1"
  if __fzf_orig_completion_get_orig_func "$cmd"; then
    "$REPLY" "$@"
  elif [[ -n "${_fzf_completion_loader-}" ]]; then
    orig_complete=$(complete -p "$orig_cmd" 2> /dev/null)
    $_fzf_completion_loader "$@"
    ret=$?
    # _completion_loader may not have updated completion for the command
    if [[ "$(complete -p "$orig_cmd" 2> /dev/null)" != "$orig_complete" ]]; then
      __fzf_orig_completion < <(complete -p "$orig_cmd" 2> /dev/null)

      # Update orig_complete by _fzf_orig_completion entry
      [[ $orig_complete =~ ' -F '(_fzf_[^ ]+)' ' ]] &&
        __fzf_orig_completion_instantiate "$cmd" "${BASH_REMATCH[1]}" &&
        orig_complete=$REPLY

      if [[ "${__fzf_nospace_commands-}" = *" $orig_cmd "* ]]; then
        eval "${orig_complete/ -F / -o nospace -F }"
      else
        eval "$orig_complete"
      fi
    fi
    [[ $ret -eq 0 ]] && return 124
    return $ret
  fi
}

__fzf_generic_path_completion() {
  local cur base dir leftover matches trigger cmd
  cmd="${COMP_WORDS[0]}"
  if [[ $cmd == \\* ]]; then
    cmd="${cmd:1}"
  fi
  COMPREPLY=()
  trigger=${FZF_COMPLETION_TRIGGER-'**'}
  cur="${COMP_WORDS[COMP_CWORD]}"
  if [[ "$cur" == *"$trigger" ]] && [[ $cur != *'$('* ]] && [[ $cur != *':='* ]] && [[ $cur != *'`'* ]]; then
    base=${cur:0:${#cur}-${#trigger}}
    eval "base=$base" 2> /dev/null || return

    dir=
    [[ $base = *"/"* ]] && dir="$base"
    while true; do
      if [[ -z "$dir" ]] || [[ -d "$dir" ]]; then
        leftover=${base/#"$dir"}
        leftover=${leftover/#\/}
        [[ -z "$dir" ]] && dir='.'
        [[ "$dir" != "/" ]] && dir="${dir/%\//}"
        matches=$(
          export FZF_DEFAULT_OPTS=$(__fzf_defaults "--reverse --scheme=path" "${FZF_COMPLETION_OPTS-} $2")
          unset FZF_DEFAULT_COMMAND FZF_DEFAULT_OPTS_FILE
          if declare -F "$1" > /dev/null; then
            eval "$1 $(printf %q "$dir")" | __fzf_comprun "$4" -q "$leftover"
          else
            if [[ $1 =~ dir ]]; then
              walker=dir,follow
              rest=${FZF_COMPLETION_DIR_OPTS-}
            else
              walker=file,dir,follow,hidden
              rest=${FZF_COMPLETION_PATH_OPTS-}
            fi
            __fzf_comprun "$4" -q "$leftover" --walker "$walker" --walker-root="$dir" $rest
          fi | while read -r item; do
            printf "%q " "${item%$3}$3"
          done
        )
        matches=${matches% }
        [[ -z "$3" ]] && [[ "${__fzf_nospace_commands-}" = *" ${COMP_WORDS[0]} "* ]] && matches="$matches "
        if [[ -n "$matches" ]]; then
          COMPREPLY=( "$matches" )
        else
          COMPREPLY=( "$cur" )
        fi
        printf '\e[5n'
        return 0
      fi
      dir=$(command dirname "$dir")
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
  post="$(caller 0 | command awk '{print $2}')_post"
  type -t "$post" > /dev/null 2>&1 || post='command cat'

  trigger=${FZF_COMPLETION_TRIGGER-'**'}
  cmd="${COMP_WORDS[0]}"
  cur="${COMP_WORDS[COMP_CWORD]}"
  if [[ "$cur" == *"$trigger" ]] && [[ $cur != *'$('* ]] && [[ $cur != *':='* ]] && [[ $cur != *'`'* ]]; then
    cur=${cur:0:${#cur}-${#trigger}}

    selected=$(
      FZF_DEFAULT_OPTS=$(__fzf_defaults "--reverse" "${FZF_COMPLETION_OPTS-} $str_arg") \
      FZF_DEFAULT_OPTS_FILE='' \
        __fzf_comprun "${rest[0]}" "${args[@]}" -q "$cur" | $post | command tr '\n' ' ')
    selected=${selected% } # Strip trailing space not to repeat "-o nospace"
    if [[ -n "$selected" ]]; then
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
  _fzf_proc_completion "$@"
}

_fzf_proc_completion() {
  _fzf_complete -m --header-lines=1 --no-preview --wrap -- "$@" < <(
    command ps -eo user,pid,ppid,start,time,command 2> /dev/null ||
      command ps -eo user,pid,ppid,time,args # For BusyBox
  )
}

_fzf_proc_completion_post() {
  command awk '{print $2}'
}

# To use custom hostname lists, override __fzf_list_hosts.
# The function is expected to print hostnames, one per line as well as in the
# desired sorting and with any duplicates removed, to standard output.
#
# e.g.
#   # Use bash-completions’s _known_hosts_real() for getting the list of hosts
#   __fzf_list_hosts() {
#     # Set the local attribute for any non-local variable that is set by _known_hosts_real()
#     local COMPREPLY=()
#     _known_hosts_real ''
#     printf '%s\n' "${COMPREPLY[@]}" | command sort -u --version-sort
#   }
if ! declare -F __fzf_list_hosts > /dev/null; then
  __fzf_list_hosts() {
    command cat <(command tail -n +1 ~/.ssh/config ~/.ssh/config.d/* /etc/ssh/ssh_config 2> /dev/null | command grep -i '^\s*host\(name\)\? ' | command awk '{for (i = 2; i <= NF; i++) print $1 " " $i}' | command grep -v '[*?%]') \
      <(command grep -oE '^[[a-z0-9.,:-]+' ~/.ssh/known_hosts 2> /dev/null | command tr ',' '\n' | command tr -d '[' | command awk '{ print $1 " " $1 }') \
      <(command grep -v '^\s*\(#\|$\)' /etc/hosts 2> /dev/null | command grep -Fv '0.0.0.0' | command sed 's/#.*//') |
      command awk '{for (i = 2; i <= NF; i++) print $i}' | command sort -u
  }
fi

_fzf_host_completion() {
  _fzf_complete +m -- "$@" < <(__fzf_list_hosts)
}

# Values for $1 $2 $3 are described here
# https://www.gnu.org/software/bash/manual/html_node/Programmable-Completion.html
# > the first argument ($1) is the name of the command whose arguments are being completed,
# > the second argument ($2) is the word being completed,
# > and the third argument ($3) is the word preceding the word being completed on the current command line.
_fzf_complete_ssh() {
  case $3 in
    -i|-F|-E)
      _fzf_path_completion "$@"
      ;;
    *)
      local user=
      [[ "$2" =~ '@' ]] && user="${2%%@*}@"
      _fzf_complete +m -- "$@" < <(__fzf_list_hosts | command awk -v user="$user" '{print user $0}')
      ;;
  esac
}

_fzf_var_completion() {
  _fzf_complete -m -- "$@" < <(
    declare -xp | command sed -En 's|^declare [^ ]+ ([^=]+).*|\1|p'
  )
}

_fzf_alias_completion() {
  _fzf_complete -m -- "$@" < <(
    alias | command sed -En 's|^alias ([^=]+).*|\1|p'
  )
}

# fzf options
complete -o default -F _fzf_opts_completion fzf
# fzf-tmux is a thin fzf wrapper that has only a few more options than fzf
# itself. As a quick improvement we take fzf's completion. Adding the few extra
# fzf-tmux specific options (like `-w WIDTH`) are left as a future patch.
complete -o default -F _fzf_opts_completion fzf-tmux

d_cmds="${FZF_COMPLETION_DIR_COMMANDS-cd pushd rmdir}"

# NOTE: $FZF_COMPLETION_PATH_COMMANDS and $FZF_COMPLETION_VAR_COMMANDS are
# undocumented and subject to change in the future.
a_cmds="${FZF_COMPLETION_PATH_COMMANDS-"
  awk bat cat code diff diff3
  emacs emacsclient ex file ftp g++ gcc gvim head hg hx java
  javac ld less more mvim nvim patch perl python ruby
  sed sftp sort source tail tee uniq vi view vim wc xdg-open
  basename bunzip2 bzip2 chmod chown curl cp dirname du
  find git grep gunzip gzip hg jar
  ln ls mv open rm rsync scp
  svn tar unzip zip"}"
v_cmds="${FZF_COMPLETION_VAR_COMMANDS-export unset printenv}"

# Preserve existing completion
__fzf_orig_completion < <(complete -p $d_cmds $a_cmds $v_cmds unalias kill ssh 2> /dev/null)

if type _comp_load > /dev/null 2>&1; then
  # _comp_load was added in bash-completion 2.12 to replace _completion_loader.
  # We use it without -D option so that it does not use _comp_complete_minimal as the fallback.
  _fzf_completion_loader=_comp_load
elif type __load_completion > /dev/null 2>&1; then
  # In bash-completion 2.11, _completion_loader internally calls __load_completion
  # and if it returns a non-zero status, it sets the default 'minimal' completion.
  _fzf_completion_loader=__load_completion
elif type _completion_loader > /dev/null 2>&1; then
  _fzf_completion_loader=_completion_loader
fi

__fzf_defc() {
  local cmd func opts REPLY
  cmd="$1"
  func="$2"
  opts="$3"
  if __fzf_orig_completion_instantiate "$cmd" "$func"; then
    eval "$REPLY"
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
  __fzf_defc "$cmd" _fzf_dir_completion "-o bashdefault -o nospace -o dirnames"
done

# Variables
for cmd in $v_cmds; do
  __fzf_defc "$cmd" _fzf_var_completion "-o default -o nospace -v"
done

# Aliases
__fzf_defc unalias _fzf_alias_completion "-a"

# Processes
__fzf_defc kill _fzf_proc_completion "-o default -o bashdefault"

# ssh
__fzf_defc ssh _fzf_complete_ssh "-o default -o bashdefault"

unset cmd d_cmds a_cmds v_cmds

_fzf_setup_completion() {
  local kind fn cmd
  kind=$1
  fn=_fzf_${1}_completion
  if [[ $# -lt 2 ]] || ! type -t "$fn" > /dev/null; then
    echo "usage: ${FUNCNAME[0]} path|dir|var|alias|host|proc COMMANDS..."
    return 1
  fi
  shift
  __fzf_orig_completion < <(complete -p "$@" 2> /dev/null)
  for cmd in "$@"; do
    case "$kind" in
      dir)   __fzf_defc "$cmd" "$fn" "-o nospace -o dirnames" ;;
      var)   __fzf_defc "$cmd" "$fn" "-o default -o nospace -v" ;;
      alias) __fzf_defc "$cmd" "$fn" "-a" ;;
      *)     __fzf_defc "$cmd" "$fn" "-o default -o bashdefault" ;;
    esac
  done
}

fi
