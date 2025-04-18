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
# For compatibility with fish versions down to 3.1.2, the script does not use:
# - The -f/--function switch of command: set
# - The process substitution syntax: $(cmd)
# - Ranges that omit start/end indexes: $var[$start..] $var[..$end] $var[..]
function fzf_key_bindings

  function __fzf_defaults
    # $argv[1]: Prepend to FZF_DEFAULT_OPTS_FILE and FZF_DEFAULT_OPTS
    # $argv[2..]: Append to FZF_DEFAULT_OPTS_FILE and FZF_DEFAULT_OPTS
    test -n "$FZF_TMUX_HEIGHT"; or set -l FZF_TMUX_HEIGHT 40%
    string join ' ' -- \
      "--height $FZF_TMUX_HEIGHT --min-height=20+ --bind=ctrl-z:ignore" $argv[1] \
      (test -r "$FZF_DEFAULT_OPTS_FILE"; and string join -- ' ' <$FZF_DEFAULT_OPTS_FILE) \
      $FZF_DEFAULT_OPTS $argv[2..-1]
  end

  function __fzfcmd
    test -n "$FZF_TMUX_HEIGHT"; or set -l FZF_TMUX_HEIGHT 40%
    if test -n "$FZF_TMUX_OPTS"
      echo "fzf-tmux $FZF_TMUX_OPTS -- "
    else if test "$FZF_TMUX" = "1"
      echo "fzf-tmux -d$FZF_TMUX_HEIGHT -- "
    else
      echo "fzf"
    end
  end

  function __fzf_parse_commandline -d 'Parse the current command line token and return split of existing filepath, fzf query, and optional -option= prefix'
    set -l fzf_query ''
    set -l prefix ''
    set -l dir '.'

    # Set variables containing the major and minor fish version numbers, using
    # a method compatible with all supported fish versions.
    set -l -- fish_major (string match -r -- '^\d+' $version)
    set -l -- fish_minor (string match -r -- '^\d+\.(\d+)' $version)[2]

    # fish v3.3.0 and newer: Don't use option prefix if " -- " is preceded.
    set -l -- match_regex '(?<fzf_query>[\s\S]*?(?=\n?$)$)'
    set -l -- prefix_regex '^-[^\s=]+=|^-(?!-)\S'
    if test "$fish_major" -eq 3 -a "$fish_minor" -lt 3
    or string match -q -v -- '* -- *' (string sub -l (commandline -Cp) -- (commandline -p))
      set -- match_regex "(?<prefix>$prefix_regex)?$match_regex"
    end

    # Set $prefix and expanded $fzf_query with preserved trailing newlines.
    if test "$fish_major" -ge 4
      # fish v4.0.0 and newer
      string match -q -r -- $match_regex (commandline --current-token --tokens-expanded | string collect -N)
    else if test "$fish_major" -eq 3 -a "$fish_minor" -ge 2
      # fish v3.2.0 - v3.7.1 (last v3)
      string match -q -r -- $match_regex (commandline --current-token --tokenize | string collect -N)
      eval set -- fzf_query (string escape -n -- $fzf_query | string replace -r -a '^\\\(?=~)|\\\(?=\$\w)' '')
    else
      # fish older than v3.2.0 (v3.1b1 - v3.1.2)
      set -l -- cl_token (commandline --current-token --tokenize | string collect -N)
      set -- prefix (string match -r -- $prefix_regex $cl_token)
      set -- fzf_query (string replace -- "$prefix" '' $cl_token | string collect -N)
      eval set -- fzf_query (string escape -n -- $fzf_query | string replace -r -a '^\\\(?=~)|\\\(?=\$\w)|\\\n\\\n$' '')
    end

    if test -n "$fzf_query"
      # Normalize path in $fzf_query, set $dir to the longest existing directory.
      if test \( "$fish_major" -ge 4 \) -o \( "$fish_major" -eq 3 -a "$fish_minor" -ge 5 \)
        # fish v3.5.0 and newer
        set -- fzf_query (path normalize -- $fzf_query)
        set -- dir $fzf_query
        while not path is -d $dir
          set -- dir (path dirname $dir)
        end
      else
        # fish older than v3.5.0 (v3.1b1 - v3.4.1)
        if test "$fish_major" -eq 3 -a "$fish_minor" -ge 2
          # fish v3.2.0 - v3.4.1
          string match -q -r -- '(?<fzf_query>^[\s\S]*?(?=\n?$)$)' \
            (string replace -r -a -- '(?<=/)/|(?<!^)/+(?!\n)$' '' $fzf_query | string collect -N)
        else
          # fish v3.1b1 - v3.1.2
          set -- fzf_query (string replace -r -a -- '(?<=/)/|(?<!^)/+(?!\n)$' '' $fzf_query | string collect -N)
          eval set -- fzf_query (string escape -n -- $fzf_query | string replace -r '\\\n$' '')
        end
        set -- dir $fzf_query
        while not test -d "$dir"
          set -- dir (dirname -z -- "$dir" | string split0)
        end
      end

      if not string match -q -- '.' $dir; or string match -q -r -- '^\./|^\.$' $fzf_query
        # Strip $dir from $fzf_query - preserve trailing newlines.
        if test "$fish_major" -ge 4
          # fish v4.0.0 and newer
          string match -q -r -- '^'(string escape --style=regex -- $dir)'/?(?<fzf_query>[\s\S]*)' $fzf_query
        else if test "$fish_major" -eq 3 -a "$fish_minor" -ge 2
          # fish v3.2.0 - v3.7.1 (last v3)
          string match -q -r -- '^/?(?<fzf_query>[\s\S]*?(?=\n?$)$)' \
            (string replace -- "$dir" '' $fzf_query | string collect -N)
        else
          # fish older than v3.2.0 (v3.1b1 - v3.1.2)
          set -- fzf_query (string replace -- "$dir" '' $fzf_query | string collect -N)
          eval set -- fzf_query (string escape -n -- $fzf_query | string replace -r -a '^/?|\\\n$' '')
        end
      end
    end

    string escape -n -- "$dir" "$fzf_query" "$prefix"
  end

  # Store current token in $dir as root for the 'find' command
  function fzf-file-widget -d "List files and folders"
    set -l commandline (__fzf_parse_commandline)
    set -lx dir $commandline[1]
    set -l fzf_query $commandline[2]
    set -l prefix $commandline[3]

    set -lx FZF_DEFAULT_OPTS (__fzf_defaults \
      "--reverse --walker=file,dir,follow,hidden --scheme=path" \
      "$FZF_CTRL_T_OPTS --multi --print0")

    set -lx FZF_DEFAULT_COMMAND "$FZF_CTRL_T_COMMAND"
    set -lx FZF_DEFAULT_OPTS_FILE

    if set -l result (eval (__fzfcmd) --walker-root=$dir --query=$fzf_query | string split0)
      # Remove last token from commandline.
      commandline -t ''
      for i in $result
        commandline -it -- $prefix(string escape -- $i)' '
      end
    end

    commandline -f repaint
  end

  function fzf-history-widget -d "Show command history"
    set -l fzf_query (commandline | string escape)

    set -lx FZF_DEFAULT_OPTS (__fzf_defaults '' \
      '--nth=2..,.. --scheme=history --multi --wrap-sign="\t↳ "' \
      "--bind=ctrl-r:toggle-sort --highlight-line $FZF_CTRL_R_OPTS" \
      '--accept-nth=2.. --read0 --print0 --with-shell='(status fish-path)\\ -c)

    set -lx FZF_DEFAULT_OPTS_FILE
    set -lx FZF_DEFAULT_COMMAND

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

    # Merge history from other sessions before searching
    test -z "$fish_private_mode"; and builtin history merge

    if set -l result (eval $FZF_DEFAULT_COMMAND \| (__fzfcmd) --query=$fzf_query | string split0)
      commandline -- (string replace -a -- \n\t \n $result[1])
      test (count $result) -gt 1; and for i in $result[2..-1]
        commandline -i -- (string replace -a -- \n\t \n \n$i)
      end
    end

    commandline -f repaint
  end

  function fzf-cd-widget -d "Change directory"
    set -l commandline (__fzf_parse_commandline)
    set -lx dir $commandline[1]
    set -l fzf_query $commandline[2]
    set -l prefix $commandline[3]

    set -lx FZF_DEFAULT_OPTS (__fzf_defaults \
      "--reverse --walker=dir,follow,hidden --scheme=path" \
      "$FZF_ALT_C_OPTS --no-multi --print0")

    set -lx FZF_DEFAULT_OPTS_FILE
    set -lx FZF_DEFAULT_COMMAND "$FZF_ALT_C_COMMAND"

    if set -l result (eval (__fzfcmd) --query=$fzf_query --walker-root=$dir | string split0)
      cd -- $result
      commandline -rt -- $prefix
    end

    commandline -f repaint
  end

  bind \cr fzf-history-widget
  bind -M insert \cr fzf-history-widget

  if not set -q FZF_CTRL_T_COMMAND; or test -n "$FZF_CTRL_T_COMMAND"
    bind \ct fzf-file-widget
    bind -M insert \ct fzf-file-widget
  end

  if not set -q FZF_ALT_C_COMMAND; or test -n "$FZF_ALT_C_COMMAND"
    bind \ec fzf-cd-widget
    bind -M insert \ec fzf-cd-widget
  end

end
