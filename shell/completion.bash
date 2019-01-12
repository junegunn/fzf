#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/-completion.bash
#
# - $FZF_TMUX                         (default: 0)
# - $FZF_TMUX_HEIGHT                  (default: '40%')
# - $FZF_COMPLETION_OPTS              (default: empty)
# - $LINES                            (default: '40')

__fzf_complete_init_vars() {
  : "${FZF_TMUX:=0}"
  : "${FZF_TMUX_HEIGHT:='40%'}"
  : "${FZF_COMPLETION_OPTS:=}"
  : "${LINES:=40}"
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

  for completion in ${source_path}/bash_completion.d/*.bash ; do
    source ${completion}
  done

}

_fzf_complete
