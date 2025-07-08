set -u
PS1= PROMPT_COMMAND= HISTFILE= HISTSIZE=100
unset <%= UNSETS.join(' ') %>
unset $(env | sed -n /^_fzf_orig/s/=.*//p)
unset $(declare -F | sed -n "/_fzf/s/.*-f //p")

export FZF_DEFAULT_OPTS="--no-scrollbar --pointer '>' --marker '>'"

# Setup fzf
# ---------
if [[ ! "$PATH" == *<%= BASE %>/bin* ]]; then
  export PATH="${PATH:+${PATH}:}<%= BASE %>/bin"
fi

# Auto-completion
# ---------------
[[ $- == *i* ]] && source "<%= BASE %>/shell/completion.<%= __method__ %>" 2> /dev/null

# Key bindings
# ------------
source "<%= BASE %>/shell/key-bindings.<%= __method__ %>"

# Old API
_fzf_complete_f() {
  _fzf_complete "+m --multi --prompt \"prompt-f> \"" "$@" < <(
    echo foo
    echo bar
  )
}

# New API
_fzf_complete_g() {
  _fzf_complete +m --multi --prompt "prompt-g> " -- "$@" < <(
    echo foo
    echo bar
  )
}

_fzf_complete_f_post() {
  awk '{print "f" $0 $0}'
}

_fzf_complete_g_post() {
  awk '{print "g" $0 $0}'
}

[ -n "${BASH-}" ] && complete -F _fzf_complete_f -o default -o bashdefault f
[ -n "${BASH-}" ] && complete -F _fzf_complete_g -o default -o bashdefault g

_comprun() {
  local command=$1
  shift

  case "$command" in
    f) fzf "$@" --preview 'echo preview-f-{}' ;;
    g) fzf "$@" --preview 'echo preview-g-{}' ;;
    *) fzf "$@" ;;
  esac
}
