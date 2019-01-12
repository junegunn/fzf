_filedir()
{
    local IFS=$'\n'

    _tilde "$cur" || return

    local fzf dir matches

    fzf="$(__fzfcmd_complete)"
    
    dir="${cur/#\~/$HOME}"
    [[ ! "${dir}" =~ "/" ]] && dir="./${dir}"
    leftover=${cur##*/}
    dir="${dir%/*}/"

    # Files asked (with eventually an extension filter)
    if [[ "$1" != -d ]]; then
      #  Munge xspec to contain uppercase version too
      # http://thread.gmane.org/gmane.comp.shells.bash.bugs/15294/focus=15306
      local xspec=${1:+"!*.@($1|${1^^})"}
      matches=$(
        _fzf_compgen_path $(printf %q "$dir") \
        | (shopt -s extglob; while read -r line; do if [[ "${line}" != ${xspec} ]];then echo "${line}"; fi;done) \
        | FZF_DEFAULT_OPTS="--height ${FZF_TMUX_HEIGHT} --reverse $FZF_DEFAULT_OPTS $FZF_COMPLETION_OPTS" \
          $fzf -m -q "${leftover}" \
        | while read -r item; do printf "%s " "$item"; done \
      )
    else
    # Only directories
      matches=$(
        _fzf_compgen_dir $(printf %q "$dir") \
        | FZF_DEFAULT_OPTS="--height ${FZF_TMUX_HEIGHT} --reverse $FZF_DEFAULT_OPTS $FZF_COMPLETION_OPTS" \
          $fzf -q "${leftover}" \
        | while read -r item; do printf "%s " "$item"; done \
      )
    fi

    matches=${matches% }

    if [ -n "$matches" ]; then
      COMPREPLY=( "$matches" )
    fi

    printf '\e[5n'
    return 0
} # _filedir()
