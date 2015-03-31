# Key bindings
# ------------
function fzf_key_bindings
  # Due to a bug of fish, we cannot use command substitution,
  # so we use temporary file instead
  if [ -z "$TMPDIR" ]
    set -g TMPDIR /tmp
  end

  function __fzf_list
    command find -L . \( -path '*/\.*' -o -fstype 'dev' -o -fstype 'proc' \) -prune \
      -o -type f -print \
      -o -type d -print \
      -o -type l -print 2> /dev/null | sed 1d | cut -b3-
  end

  function __fzf_list_dir
    command find -L . \( -path '*/\.*' -o -fstype 'dev' -o -fstype 'proc' \) \
      -prune -o -type d -print 2> /dev/null | sed 1d | cut -b3-
  end

  function __fzf_escape
    while read item
      echo -n (echo -n "$item" | sed -E 's/([ "$~'\''([{<>})])/\\\\\\1/g')' '
    end
  end

  function __fzf_ctrl_t
    if [ -n "$TMUX_PANE" -a "$FZF_TMUX" != "0" ]
      # FIXME need to handle directory with double-quotes
      tmux split-window (__fzf_tmux_height) "cd \"$PWD\";fish -c 'fzf_key_bindings; __fzf_ctrl_t_tmux \\$TMUX_PANE'"
    else
      __fzf_list | fzf -m > $TMPDIR/fzf.result
      and commandline -i (cat $TMPDIR/fzf.result | __fzf_escape)
      commandline -f repaint
      rm -f $TMPDIR/fzf.result
    end
  end

  function __fzf_ctrl_t_tmux
    __fzf_list | fzf -m > $TMPDIR/fzf.result
    and tmux send-keys -t $argv[1] (cat $TMPDIR/fzf.result | __fzf_escape)
    rm -f $TMPDIR/fzf.result
  end

  function __fzf_ctrl_r
    history | fzf +s +m --toggle-sort=ctrl-r > $TMPDIR/fzf.result
    and commandline (cat $TMPDIR/fzf.result)
    commandline -f repaint
    rm -f $TMPDIR/fzf.result
  end

  function __fzf_alt_c
    # Fish hangs if the command before pipe redirects (2> /dev/null)
    __fzf_list_dir | fzf +m > $TMPDIR/fzf.result
    [ (cat $TMPDIR/fzf.result | wc -l) -gt 0 ]
    and cd (cat $TMPDIR/fzf.result)
    commandline -f repaint
    rm -f $TMPDIR/fzf.result
  end

  function __fzf_tmux_height
    if set -q FZF_TMUX_HEIGHT
      set height $FZF_TMUX_HEIGHT
    else
      set height 40%
    end
    if echo $height | \grep -q -E '%$'
      echo "-p "(echo $height | sed 's/%$//')
    else
      echo "-l $height"
    end
    set -e height
  end

  bind \ct '__fzf_ctrl_t'
  bind \cr '__fzf_ctrl_r'
  bind \ec '__fzf_alt_c'
end

