#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/-completion.bash
#
# - $FZF_TMUX                         (default: 0)
# - $FZF_TMUX_HEIGHT                  (default: '40%')
# - $FZF_COMPLETION_OPTS              (default: --select-1 --exit-0)
# - $LINES                            (default: '40')
# - $FZF_COMPLETION_COMPAT_MODE       (default: 1)
# - $FZF_COMPLETION_EXCLUDE           (default: empty)

__fzf_complete_init_vars() {
  : "${FZF_TMUX:=0}"
  : "${FZF_TMUX_HEIGHT:='40%'}"
  : "${FZF_COMPLETION_OPTS:=--select-1 --exit-0}"
  : "${LINES:=40}"
  : "${FZF_COMPLETION_COMPAT_MODE:=1}"
  : "${FZF_COMPLETION_EXCLUDE:=}"
}

###########################################################

# To use custom commands instead of find, override _fzf_compgen_{path,dir}
if ! declare -f _fzf_compgen_path > /dev/null; then
  _fzf_compgen_path() {
    command find -L "$1" \
      -name .git -prune -o -name .svn -prune -o \( -type d -o -type f -o -type l \) \
      -a -not -path "$1" -print 2> /dev/null | sed 's@^\./@@'
  }
fi

if ! declare -f _fzf_compgen_dir > /dev/null; then
  _fzf_compgen_dir() {
    command find -L "$1" \
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

_fzf_complete() {
  local source_path
  source_path="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

  __fzf_complete_init_vars

  local exclude_func=( ${FZF_COMPLETION_EXCLUDE} )

  for completion in ${source_path}/bash_completion.d/* ; do
    local basename="${completion##*/}"
    # If in compatibility mode
    if [[ "${FZF_COMPLETION_COMPAT_MODE}" == "1" ]];then
      # and not exclude
      if [[ ! " ${exclude_func[@]} " =~ " ${basename} " ]]; then
        # add fzf over if already exist or Create / replace functions
        _fzf_complete_over "${basename}" || source "${completion}"
      fi
    else
      # If not exclude
      if [[ ! " ${exclude_func[@]} " =~ " ${basename} " ]]; then
        # Create / replace functions
        source "${completion}"
      fi
    fi
  done

}

_fzf_complete_over() {
  local func="${1}"
  type -t "${1}" 2>&1 > /dev/null || return 1
  local origin=$(declare -f "${func}" | tail -n +3 | head -n -1)
  [[ "${origin}" =~ '# Supercharged with fzf' ]] && return 0
  local add_def
  IFS='' read -r -d '' add_def <<'EOF'
      : '# Supercharged with fzf'
      fzf="$(__fzfcmd_complete)"
      matches=$(
        printf "%s\n" "${COMPREPLY[@]}" \
        | FZF_DEFAULT_OPTS="--bind 'tab:accept' --height ${FZF_TMUX_HEIGHT} --reverse $FZF_DEFAULT_OPTS $FZF_COMPLETION_OPTS" \
          $fzf \
        | while read -r item; do printf "%s " "$item"; done \
      )
      matches="${matches% }"
      if [ -n "$matches" ]; then
        COMPREPLY=( "$matches" )
      fi
      printf '\e[5n'
EOF
  eval "
    ${func}() {
    ${origin}
    ${add_def}
    }
  "
}

_fzf_complete
