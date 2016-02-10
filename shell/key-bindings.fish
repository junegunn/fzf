# Key bindings
# ------------
function fzf_key_bindings
  function __fzf_escape
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
    eval "$FZF_CTRL_T_COMMAND" | __fzfcmd -m | __fzf_escape | read -l select
    and commandline -i "$select"
    commandline -f repaint
  end

  function __fzf_ctrl_r
    history | __fzfcmd +s +m --tiebreak=index --toggle-sort=ctrl-r | read -l select
    and commandline "$select"
    commandline -f repaint
  end

  function __fzf_alt_c
    set -q FZF_ALT_C_COMMAND; or set -l FZF_ALT_C_COMMAND "
    command find -L . \\( -path '*/\\.*' -o -fstype 'dev' -o -fstype 'proc' \\) -prune \
      -o -type d -print 2> /dev/null | sed 1d | cut -b3-"
    # Fish hangs if the command before pipe redirects (2> /dev/null)
    eval "$FZF_ALT_C_COMMAND" | __fzfcmd +m | read -la select
    [ (count $select) -gt 0 ]
    and cd $select
    commandline -f repaint
  end

  function __fzfcmd
    set -q FZF_TMUX; or set -l FZF_TMUX 1
    set -q FZF_TMUX_HEIGHT; or set -l FZF_TMUX_HEIGHT 40%

    if [ $FZF_TMUX -eq 1 ]
      fzf-tmux -d$FZF_TMUX_HEIGHT $argv
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
