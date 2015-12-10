# Key bindings
# ------------
function fzf_key_bindings

  function __join_lines
    paste -s -d ' ' # in the future it should be replaced by `string join " "`
  end

  function __trim
    xargs # in the future it should be replaced by `string trim`
  end

  
  function __fzf_escape # can be replaced by using quoted paths
    while read item
      echo -n (echo -n "$item" | sed -E 's/([ "$~'\''([{<>})])/\\\\\\1/g')' '
    end
  end

  function __fzf_ctrl_t
    set -q FZF_CTRL_T_COMMAND; or set -l FZF_CTRL_T_COMMAND "
    command find -L . \\( -path '*/\\.*' -o -fstype 'dev' -o -fstype 'proc' \\) -prune \
      -o -type f -print \
      -o -type d -print \
      -o -type l -print 2> /dev/null | sed 1d | cut -b3-"
    eval "$FZF_CTRL_T_COMMAND" | __fzfcmd -m | __fzf_escape | __join_lines | read -l selection
    and commandline -i $selection
    commandline -f repaint
  end

  function __fzf_ctrl_r
    set -l query (commandline|xargs)
    set -l args +s +m --tiebreak=index --toggle-sort=ctrl-r
    
    if test -n $query
        set args $args '-q' $query
    end

    history | __fzfcmd $args | read -l selection
    and commandline $selection
    commandline -f repaint
  end

  function __fzf_alt_c
    set -q FZF_ALT_C_COMMAND; or set -l FZF_ALT_C_COMMAND "
    command find -L . \\( -path '*/\\.*' -o -fstype 'dev' -o -fstype 'proc' \\) -prune \
      -o -type d -print 2> /dev/null | sed 1d | cut -b3-"
    # Fish hangs if the command before pipe redirects (2> /dev/null)
    eval "$FZF_ALT_C_COMMAND" | __fzfcmd +m | read -l selection
    test (echo $selection | wc -l) -gt 0
    and cd $selection
    commandline -f repaint
  end

  function __fzfcmd
    set -q FZF_TMUX; or set FZF_TMUX 1
    if [ $FZF_TMUX -eq 1 ]
      if set -q FZF_TMUX_HEIGHT
        fzf-tmux -d$FZF_TMUX_HEIGHT $argv
      else
        fzf-tmux -d40% $argv
      end
    else
      fzf $argv
    end
  end

  bind \ct '__fzf_ctrl_t'
  bind \cr '__fzf_ctrl_r'
  bind \ec '__fzf_alt_c'

  if bind -M insert > /dev/null 2>&1
    bind -M insert \ct '__fzf_ctrl_t'
    bind -M insert \cr '__fzf_ctrl_r'
    bind -M insert \ec '__fzf_alt_c'
  end
end

