#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/-completion.bash
#
# **** Not dynamic variables ****
# - $FZF_COMPLETION_COMPAT_MODE       (default: 1)
# - $FZF_COMPLETION_EXCLUDE           (default: empty)
# - $FZF_COMPLETION_TREE_PATH         (default: bash_completion.d)
#
# **** All modes ****
# - $FZF_TMUX                         (default: 0)
# - $FZF_TMUX_HEIGHT                  (default: '40%')
# - $FZF_COMPLETION_OPTS              (default: --select-1 --exit-0)
# - $LINES                            (default: '40')
#
# **** Only when FZF_COMPLETION_COMPAT_MODE=0 ****
# - $FZF_COMPLETION_MAXDEPTH          (default: 999999999)
# - $FZF_COMPLETION_PATH_OPTS         (default: empty)
# - $FZF_COMPLETION_DIR_OPTS          (default: empty)

__fzf_complete_init_vars() {

  : "${FZF_COMPLETION_COMPAT_MODE:=1}"
  : "${FZF_COMPLETION_EXCLUDE:=}"
  : "${FZF_COMPLETION_TREE_PATH:=$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )}"
  IFS=':' read -r -a FZF_COMPLETION_TREE_PATH_ARRAY <<< "${FZF_COMPLETION_TREE_PATH}"

  : "${FZF_TMUX:=0}"
  : "${FZF_TMUX_HEIGHT:='40%'}"
  : "${FZF_COMPLETION_OPTS:=--select-1 --exit-0}"
  : "${LINES:=40}"

  : "${FZF_COMPLETION_MAXDEPTH:=999999999}"
  : "${FZF_COMPLETION_PATH_OPTS:=}"
  : "${FZF_COMPLETION_DIR_OPTS:=}"
}

###########################################################

# To use custom commands instead of find, override _fzf_compgen_{path,dir}
if ! declare -f _fzf_compgen_path > /dev/null; then
  _fzf_compgen_path() {
    command find -L "$1" -maxdepth "${FZF_COMPLETION_MAXDEPTH}" \
      -name .git -prune -o -name .svn -prune -o \( -type d -o -type f -o -type l \) \
      -a -not -path "$1" -print 2> /dev/null | sed 's@^\./@@'
  }
fi

if ! declare -f _fzf_compgen_dir > /dev/null; then
  _fzf_compgen_dir() {
    command find -L "$1" -maxdepth "${FZF_COMPLETION_MAXDEPTH}" \
      -name .git -prune -o -name .svn -prune -o -type d \
      -a -not -path "$1" -print 2> /dev/null | sed 's@^\./@@'
  }
fi

###########################################################

# To redraw line after fzf closes (printf '\e[5n')
bind '"\e[0n": redraw-current-line'

__fzfcmd_complete() {
  [ -n "$TMUX_PANE" ] && [ "${FZF_TMUX}" != 0 ] && [ ${LINES} -gt 15 ] &&
    echo "fzf-tmux -d${FZF_TMUX_HEIGHT}" || echo "fzf"
}

__fzf_complete_loaders() {
  # Some completion functions are dynamically loaded (example: git)
  # So load them now to allow surcharge / erase after
  local path loader
  for path in "${FZF_COMPLETION_TREE_PATH_ARRAY[@]}";do
    for loader in ${path}/bash_completion.d/loaders/* ; do
      [ -e "${loader}" ] || continue
      source "${loader}"
    done
  done
}

__fzf_complete_traps() {
  # Traps are only used in compatibility mode
  if [[ "${FZF_COMPLETION_COMPAT_MODE}" == "1" ]];then
    local path trap
    for path in "${FZF_COMPLETION_TREE_PATH_ARRAY[@]}";do
      for trap in ${path}/bash_completion.d/traps/* ; do
        [ -e "${trap}" ] || continue
        source "${trap}"
      done
    done
  fi
}

__fzf_complete_functions() {
  local exclude_func=( ${FZF_COMPLETION_EXCLUDE} )

  local path function
  for path in "${FZF_COMPLETION_TREE_PATH_ARRAY[@]}";do
    for function in ${path}/bash_completion.d/functions/* ; do
      [ -e "${function}" ] || continue
      local basename="${function##*/}"
      # If in compatibility mode
      if [[ "${FZF_COMPLETION_COMPAT_MODE}" == "1" ]];then
        # and not exclude
        if [[ ! " ${exclude_func[@]} " =~ " ${basename} " ]]; then
          # add fzf over if already exist or create function if not
          _fzf_complete_over "${basename}" || source "${function}"
        fi
      else
        # If not excluded
        if [[ ! " ${exclude_func[@]} " =~ " ${basename} " ]]; then
          if [[ -s "${function}" ]];then
            # Create / replace functions
            source "${function}"
          else
            # Add compatibility mode if file is empty
            _fzf_complete_over "${basename}"
          fi
        fi
      fi
    done
  done
}

_fzf_complete() {
  __fzf_complete_init_vars
  __fzf_complete_loaders
  __fzf_complete_traps
  __fzf_complete_functions
}

_fzf_complete_over() {
  local func="${1}"
  type -t "${func}" 2>&1 > /dev/null || return 1
  local origin=$(declare -f "${func}" | tail -n +3 | head -n -1)
  [[ "${origin}" =~ '_fzf_complete_trap' ]] && return 0
  local add_def

  local trap='_fzf_complete_trap'

  # If a specific trap exist, use it
  if type -t "_fzf_complete_trap${func}" 2>&1 > /dev/null;then
    trap="_fzf_complete_trap${func}"
  fi

  add_def='trap '"'"''${trap}' "$?" "${COMPREPLY[@]}"; trap - RETURN'"'"' RETURN'

  eval "
  ${func}() {
    ${add_def}
    ${origin}
  }
  "
}

_fzf_complete_trap() {
  local status=$1
  shift

  [[ ${status} != 0 ]] && return ${status}

  local array=("$@")
  local fzf="$(__fzfcmd_complete)"
  if [[ ${#array[@]} -ne 0 ]];then
    local matches=$(
      printf "%s\n" "${array[@]}" | sort -u \
      | FZF_DEFAULT_OPTS="--bind 'tab:accept' --height ${FZF_TMUX_HEIGHT} --reverse $FZF_DEFAULT_OPTS $FZF_COMPLETION_OPTS" \
        $fzf \
      | while read -r item; do printf "%s" "${item}"; done \
    )
    if [ -n "$matches" ]; then
      COMPREPLY=( "${matches}" )
    else
      COMPREPLY=()
    fi
    compopt +o nospace
    printf '\e[5n'
  fi
}

_fzf_complete
