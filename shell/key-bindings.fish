#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ key-bindings.fish
#
# - $FZF_TMUX_OPTS
# - $FZF_CTRL_T_COMMAND
# - $FZF_CTRL_T_OPTS
# - $FZF_CTRL_R_OPTS
# - $FZF_ALT_C_COMMAND
# - $FZF_ALT_C_OPTS


# Key bindings
# ------------
function fzf_key_bindings

  function __fzf_defaults
    # $1: Prepend to FZF_DEFAULT_OPTS_FILE and FZF_DEFAULT_OPTS
    # $2: Append to FZF_DEFAULT_OPTS_FILE and FZF_DEFAULT_OPTS
    test -n "$FZF_TMUX_HEIGHT"; or set FZF_TMUX_HEIGHT 40%
    echo "--height $FZF_TMUX_HEIGHT --min-height 20+ --bind=ctrl-z:ignore" $argv[1]
    test -r "$FZF_DEFAULT_OPTS_FILE"; and string collect -N -- <$FZF_DEFAULT_OPTS_FILE
    echo $FZF_DEFAULT_OPTS $argv[2]
  end

  # Store current token in $dir as root for the 'find' command
  function fzf-file-widget -d "List files and folders"
    set -l commandline (__fzf_parse_commandline)
    set -lx dir $commandline[1]
    set -l fzf_query $commandline[2]
    set -l prefix $commandline[3]
    set -l result

    test -n "$FZF_TMUX_HEIGHT"; or set FZF_TMUX_HEIGHT 40%
    begin
      set -lx FZF_DEFAULT_OPTS (__fzf_defaults "--reverse --walker=file,dir,follow,hidden --scheme=path --walker-root=$dir" "$FZF_CTRL_T_OPTS")
      set -lx FZF_DEFAULT_COMMAND "$FZF_CTRL_T_COMMAND"
      set -lx FZF_DEFAULT_OPTS_FILE ''
      set result (eval (__fzfcmd) -m --query=$fzf_query)
    end
    if test -z "$result"
      commandline -f repaint
      return
    else
      # Remove last token from commandline.
      commandline -t ""
    end
    for i in $result
      commandline -it -- $prefix
      commandline -it -- (string escape -- $i)
      commandline -it -- ' '
    end
    commandline -f repaint
  end

  function fzf-history-widget -d "Show command history"
    test -n "$FZF_TMUX_HEIGHT"; or set FZF_TMUX_HEIGHT 40%
    begin
      # merge history from other sessions before searching
      test -z "$fish_private_mode"; and builtin history merge

      set -lx FZF_DEFAULT_OPTS (__fzf_defaults "" "-n2..,.. --scheme=history --bind=ctrl-r:toggle-sort --wrap-sign '"\t"↳ ' --highlight-line +m $FZF_CTRL_R_OPTS")
      set -lx FZF_DEFAULT_OPTS_FILE ''
      set -lx FZF_DEFAULT_COMMAND
      set -a -- FZF_DEFAULT_OPTS --with-shell=(status fish-path)\\ -c

      if type -q perl
        set -a FZF_DEFAULT_OPTS '--tac'
        set FZF_DEFAULT_COMMAND 'builtin history -z --reverse | command perl -0 -pe \'s/^/$.\t/g; s/\n/\n\t/gm\''
      else
        set FZF_DEFAULT_COMMAND \
          'set -l h (builtin history -z --reverse | string split0);' \
          'for i in (seq (count $h) -1 1);' \
          'string join0 -- $i\t(string replace -a -- \n \n\t $h[$i] | string collect);' \
          'end'
      end
      set -l result (eval $FZF_DEFAULT_COMMAND \| (__fzfcmd) --read0 --print0 -q (commandline | string escape) "--bind=enter:become:'string replace -a -- \n\t \n {2..} | string collect'")
      and commandline -- $result
    end
    commandline -f repaint
  end

  function fzf-cd-widget -d "Change directory"
    set -l commandline (__fzf_parse_commandline)
    set -lx dir $commandline[1]
    set -l fzf_query $commandline[2]
    set -l prefix $commandline[3]

    test -n "$FZF_TMUX_HEIGHT"; or set FZF_TMUX_HEIGHT 40%
    begin
      set -lx FZF_DEFAULT_OPTS (__fzf_defaults "--reverse --walker=dir,follow,hidden --scheme=path --walker-root=$dir" "$FZF_ALT_C_OPTS")
      set -lx FZF_DEFAULT_OPTS_FILE ''
      set -lx FZF_DEFAULT_COMMAND "$FZF_ALT_C_COMMAND"
      set -l result (eval (__fzfcmd) +m --query=$fzf_query)

      if test -n "$result"
        cd -- $result

        # Remove last token from commandline.
        commandline -t ""
        commandline -it -- $prefix
      end
    end

    commandline -f repaint
  end

  function __fzfcmd
    test -n "$FZF_TMUX"; or set FZF_TMUX 0
    test -n "$FZF_TMUX_HEIGHT"; or set FZF_TMUX_HEIGHT 40%
    if test -n "$FZF_TMUX_OPTS"
      echo "fzf-tmux $FZF_TMUX_OPTS -- "
    else if test "$FZF_TMUX" = "1"
      echo "fzf-tmux -d$FZF_TMUX_HEIGHT -- "
    else
      echo "fzf"
    end
  end

  bind \cr fzf-history-widget
  if not set -q FZF_CTRL_T_COMMAND; or test -n "$FZF_CTRL_T_COMMAND"
    bind \ct fzf-file-widget
  end
  if not set -q FZF_ALT_C_COMMAND; or test -n "$FZF_ALT_C_COMMAND"
    bind \ec fzf-cd-widget
  end

  bind -M insert \cr fzf-history-widget
  if not set -q FZF_CTRL_T_COMMAND; or test -n "$FZF_CTRL_T_COMMAND"
    bind -M insert \ct fzf-file-widget
  end
  if not set -q FZF_ALT_C_COMMAND; or test -n "$FZF_ALT_C_COMMAND"
    bind -M insert \ec fzf-cd-widget
  end

  function __fzf_parse_commandline -d 'Parse the current command line token and return split of existing filepath, fzf query, and optional -option= prefix'
    set -l commandline (commandline -t)

    # strip -option= from token if present
    set -l prefix (string match -r -- '^-[^\s=]+=' $commandline)
    set commandline (string replace -- "$prefix" '' $commandline)

    # Enable home directory expansion of leading ~/
    set commandline (string replace -r -- '^~/' '\$HOME/' $commandline)

    # escape special characters, except for the $ sign of valid variable names,
    # so that after eval, the original string is returned, but with the
    # variable names replaced by their values.
    set commandline (string escape -n -- $commandline)
    set commandline (string replace -r -a -- '\x5c\$(?=[\w])' '\$' $commandline)

    # eval is used to do shell expansion on paths
    eval set commandline $commandline

    # Combine multiple consecutive slashes into one
    set commandline (string replace -r -a -- '/+' '/' $commandline)

    if test -z "$commandline"
      # Default to current directory with no --query
      set dir '.'
      set fzf_query ''
    else
      set dir (__fzf_get_dir $commandline)

      # BUG: on combined expressions, if a left argument is a single `!`, the
      # builtin test command of fish will treat it as the ! operator. To
      # overcome this, have the variable parts on the right.
      if test "." = "$dir" -a "./" != (string sub -l 2 -- $commandline)
        # if $dir is "." but commandline is not a relative path, this means no file path found
        set fzf_query $commandline
      else
        # Also remove trailing slash after dir, to "split" input properly
        set fzf_query (string replace -r -- "^$dir/?" '' $commandline)
      end
    end

    echo (string escape -- $dir)
    echo (string escape -- $fzf_query)
    echo $prefix
  end

  function __fzf_get_dir -d 'Find the longest existing filepath from input string'
    set dir $argv

    # Strip trailing slash, unless $dir is root dir (/)
    set dir (string replace -r -- '(?<!^)/$' '' $dir)

    # Iteratively check if dir exists and strip tail end of path
    while test ! -d "$dir"
      # If path is absolute, this can keep going until ends up at /
      # If path is relative, this can keep going until entire input is consumed, dirname returns "."
      set dir (dirname -- "$dir")
    end

    echo $dir
  end

end
