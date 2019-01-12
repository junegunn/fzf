_kill()
{
    local cur prev words cword
    _init_completion || return

    case $prev in
        -s)
            _signals
            return
            ;;
        -l)
            return
            ;;
    esac

    if [[ $cword -eq 1 && "$cur" == -* ]]; then
        # return list of available signals
        _signals -
        COMPREPLY+=( $( compgen -W "-s -l" -- "$cur" ) )
    else
      local selected fzf
      fzf="$(__fzfcmd_complete)"
      selected=$(
        command ps -ef \
        | sed 1d \
        | FZF_DEFAULT_OPTS="--height ${FZF_TMUX_HEIGHT} --min-height 15 --reverse $FZF_DEFAULT_OPTS --preview 'echo {}' --preview-window down:3:wrap $FZF_COMPLETION_OPTS" \
          $fzf -m \
        | awk '{print $2}' \
        | tr '\n' ' '
      )
      printf '\e[5n'

      if [ -n "$selected" ]; then
        COMPREPLY=( "$selected" )
        return 0
      fi
    fi
} &&
complete -F _kill kill
