# Key bindings
# ------------
function fzf_key_bindings

  # Store current token in $dir as root for the 'find' command
  function fzf-file-widget -d "List files and folders"
    set -l dir (commandline -t)
    # The commandline token might be escaped, we need to unescape it.
    set dir (eval "printf '%s' $dir")
    if [ ! -d "$dir" ]
      set dir .
    end
    # Some 'find' versions print undesired duplicated slashes if the path ends with slashes.
    set dir (string replace --regex '(.)/+$' '$1' "$dir")

    # "-path \$dir'*/\\.*'" matches hidden files/folders inside $dir but not
    # $dir itself, even if hidden.
    set -q FZF_CTRL_T_COMMAND; or set -l FZF_CTRL_T_COMMAND "
    command find -L \$dir -mindepth 1 \\( -path \$dir'*/\\.*' -o -fstype 'sysfs' -o -fstype 'devfs' -o -fstype 'devtmpfs' \\) -prune \
    -o -type f -print \
    -o -type d -print \
    -o -type l -print 2> /dev/null | cut -b3-"

    set -q FZF_TMUX_HEIGHT; or set FZF_TMUX_HEIGHT 40%
    begin
      set -lx FZF_DEFAULT_OPTS "--height $FZF_TMUX_HEIGHT --reverse $FZF_DEFAULT_OPTS $FZF_CTRL_T_OPTS"
      eval "$FZF_CTRL_T_COMMAND | "(__fzfcmd)" -m" | while read -l r; set result $result $r; end
    end
    if [ -z "$result" ]
      commandline -f repaint
      return
    end

    if [ "$dir" != . ]
      # Remove last token from commandline.
      commandline -t ""
    end
    for i in $result
      commandline -it -- (string escape $i)
      commandline -it -- ' '
    end
    commandline -f repaint
  end

  function fzf-history-widget -d "Show command history"
    set -q FZF_TMUX_HEIGHT; or set FZF_TMUX_HEIGHT 40%
    begin
      set -lx FZF_DEFAULT_OPTS "--height $FZF_TMUX_HEIGHT $FZF_DEFAULT_OPTS --tiebreak=index --bind=ctrl-r:toggle-sort $FZF_CTRL_R_OPTS +m"
      history | eval (__fzfcmd) -q '(commandline)' | read -l result
      and commandline -- $result
    end
    commandline -f repaint
  end

  function fzf-cd-widget -d "Change directory"
    set -q FZF_ALT_C_COMMAND; or set -l FZF_ALT_C_COMMAND "
    command find -L . -mindepth 1 \\( -path '*/\\.*' -o -fstype 'sysfs' -o -fstype 'devfs' -o -fstype 'devtmpfs' \\) -prune \
    -o -type d -print 2> /dev/null | cut -b3-"
    set -q FZF_TMUX_HEIGHT; or set FZF_TMUX_HEIGHT 40%
    begin
      set -lx FZF_DEFAULT_OPTS "--height $FZF_TMUX_HEIGHT --reverse $FZF_DEFAULT_OPTS $FZF_ALT_C_OPTS"
      eval "$FZF_ALT_C_COMMAND | "(__fzfcmd)" +m" | read -l result
      [ "$result" ]; and cd $result
    end
    commandline -f repaint
  end

  function __fzfcmd
    set -q FZF_TMUX; or set FZF_TMUX 0
    set -q FZF_TMUX_HEIGHT; or set FZF_TMUX_HEIGHT 40%
    if [ $FZF_TMUX -eq 1 ]
      echo "fzf-tmux -d$FZF_TMUX_HEIGHT"
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
