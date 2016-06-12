# Key bindings
# ------------
function fzf_key_bindings
  # Due to a bug of fish, we cannot use command substitution,
  # so we use temporary file instead
  if [ -z "$TMPDIR" ]
    set -g TMPDIR /tmp
  end

  function __fzf_escape
    while read item
      echo -n (echo -n "$item" | sed -E 's/([ "$~'\''([{<>})])/\\\\\\1/g')' '
    end
  end

  function fzf-file-widget
    set -q FZF_CTRL_T_COMMAND; or set -l FZF_CTRL_T_COMMAND "
    command find -L . \\( -path '*/\\.*' -o -fstype 'dev' -o -fstype 'proc' \\) -prune \
      -o -type f -print \
      -o -type d -print \
      -o -type l -print 2> /dev/null | sed 1d | cut -b3-"
    eval "$FZF_CTRL_T_COMMAND | "(__fzfcmd)" -m $FZF_CTRL_T_OPTS > $TMPDIR/fzf.result"
    and for i in (seq 20); commandline -i (cat $TMPDIR/fzf.result | __fzf_escape) 2> /dev/null; and break; sleep 0.1; end
    commandline -f repaint
    rm -f $TMPDIR/fzf.result
  end

  function fzf-history-widget
    history | eval (__fzfcmd) +s +m --tiebreak=index --toggle-sort=ctrl-r $FZF_CTRL_R_OPTS > $TMPDIR/fzf.result
    and commandline (cat $TMPDIR/fzf.result)
    commandline -f repaint
    rm -f $TMPDIR/fzf.result
  end

  function fzf-cd-widget
    set -q FZF_ALT_C_COMMAND; or set -l FZF_ALT_C_COMMAND "
    command find -L . \\( -path '*/\\.*' -o -fstype 'dev' -o -fstype 'proc' \\) -prune \
      -o -type d -print 2> /dev/null | sed 1d | cut -b3-"
    # Fish hangs if the command before pipe redirects (2> /dev/null)
    eval "$FZF_ALT_C_COMMAND | "(__fzfcmd)" +m $FZF_ALT_C_OPTS > $TMPDIR/fzf.result"
    [ (cat $TMPDIR/fzf.result | wc -l) -gt 0 ]
    and cd (cat $TMPDIR/fzf.result)
    commandline -f repaint
    rm -f $TMPDIR/fzf.result
  end

  function __fzfcmd
    set -q FZF_TMUX; or set FZF_TMUX 1

    if [ $FZF_TMUX -eq 1 ]
      if set -q FZF_TMUX_HEIGHT
        echo "fzf-tmux -d$FZF_TMUX_HEIGHT"
      else
        echo "fzf-tmux -d40%"
      end
    else
      echo "fzf"
    end
  end

  bind \ct fzf-file-widget
  bind \cr fzf-history-widget
  bind \ec fzf-cd-widget

  if bind -M insert > /dev/null 2>&1
    bind -M insert \ct fzf-file-widget
    bind -M insert \cr fzf-history-widget
    bind -M insert \ec fzf-cd-widget
  end
end

