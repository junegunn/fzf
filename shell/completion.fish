#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ completion.fish
#
# - $FZF_COMPLETION_OPTS                  (default: empty)

function fzf_completion_setup

#----BEGIN INCLUDE common.fish
# NOTE: Do not directly edit this section, which is copied from "common.fish".
# To modify it, one can edit "common.fish" and run "./update.sh" to apply
# the changes. See code comments in "common.fish" for the implementation details.

  function __fzf_defaults
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

  function __fzf_cmd_tokens -d 'Return command line tokens, skipping leading env assignments and command prefixes'
    set -l tokens
    if test (string match -r -- '^\d+' $version) -ge 4
      set -- tokens (commandline -xpc)
    else
      set -- tokens (commandline -opc)
    end

    set -l -- var_count 0
    for i in $tokens
      if string match -qr -- '^[\w]+=' $i
        set var_count (math $var_count + 1)
      else
        break
      end
    end
    set -e -- tokens[0..$var_count]

    while true
      switch "$tokens[1]"
        case builtin command
          set -e -- tokens[1]
          test "$tokens[1]" = "--"; and set -e -- tokens[1]
        case env
          set -e -- tokens[1]
          test "$tokens[1]" = "--"; and set -e -- tokens[1]
          while string match -qr -- '^[\w]+=' "$tokens[1]"
            set -e -- tokens[1]
          end
        case '*'
          break
      end
    end

    string escape -n -- $tokens
  end

  function __fzf_parse_commandline -d 'Parse the current command line token and return split of existing filepath, fzf query, and optional -option= prefix'
    set -l fzf_query ''
    set -l prefix ''
    set -l dir '.'

    set -l -- fish_major (string match -r -- '^\d+' $version)
    set -l -- fish_minor (string match -r -- '^\d+\.(\d+)' $version)[2]

    set -l -- match_regex '(?<fzf_query>[\s\S]*?(?=\n?$)$)'
    set -l -- prefix_regex '^-[^\s=]+=|^-(?!-)\S'
    if test "$fish_major" -eq 3 -a "$fish_minor" -lt 3
    or string match -q -v -- '* -- *' (string sub -l (commandline -Cp) -- (commandline -p))
      set -- match_regex "(?<prefix>$prefix_regex)?$match_regex"
    end

    if test "$fish_major" -ge 4
      string match -q -r -- $match_regex (commandline --current-token --tokens-expanded | string collect -N)
    else if test "$fish_major" -eq 3 -a "$fish_minor" -ge 2
      string match -q -r -- $match_regex (commandline --current-token --tokenize | string collect -N)
      eval set -- fzf_query (string escape -n -- $fzf_query | string replace -r -a '^\\\(?=~)|\\\(?=\$\w)' '')
    else
      set -l -- cl_token (commandline --current-token --tokenize | string collect -N)
      set -- prefix (string match -r -- $prefix_regex $cl_token)
      set -- fzf_query (string replace -- "$prefix" '' $cl_token | string collect -N)
      eval set -- fzf_query (string escape -n -- $fzf_query | string replace -r -a '^\\\(?=~)|\\\(?=\$\w)|\\\n\\\n$' '')
    end

    if test -n "$fzf_query"
      if test \( "$fish_major" -ge 4 \) -o \( "$fish_major" -eq 3 -a "$fish_minor" -ge 5 \)
        set -- fzf_query (path normalize -- $fzf_query)
        set -- dir $fzf_query
        while not path is -d $dir
          set -- dir (path dirname $dir)
        end
      else
        if test "$fish_major" -eq 3 -a "$fish_minor" -ge 2
          string match -q -r -- '(?<fzf_query>^[\s\S]*?(?=\n?$)$)' \
            (string replace -r -a -- '(?<=/)/|(?<!^)/+(?!\n)$' '' $fzf_query | string collect -N)
        else
          set -- fzf_query (string replace -r -a -- '(?<=/)/|(?<!^)/+(?!\n)$' '' $fzf_query | string collect -N)
          eval set -- fzf_query (string escape -n -- $fzf_query | string replace -r '\\\n$' '')
        end
        set -- dir $fzf_query
        while not test -d "$dir"
          set -- dir (dirname -z -- "$dir" | string split0)
        end
      end

      if not string match -q -- '.' $dir; or string match -q -r -- '^\./|^\.$' $fzf_query
        if test "$fish_major" -ge 4
          string match -q -r -- '^'(string escape --style=regex -- $dir)'/?(?<fzf_query>[\s\S]*)' $fzf_query
        else if test "$fish_major" -eq 3 -a "$fish_minor" -ge 2
          string match -q -r -- '^/?(?<fzf_query>[\s\S]*?(?=\n?$)$)' \
            (string replace -- "$dir" '' $fzf_query | string collect -N)
        else
          set -- fzf_query (string replace -- "$dir" '' $fzf_query | string collect -N)
          eval set -- fzf_query (string escape -n -- $fzf_query | string replace -r -a '^/?|\\\n$' '')
        end
      end
    end

    string escape -n -- "$dir" "$fzf_query" "$prefix"
  end
#----END INCLUDE

  # Use complete builtin for specific commands
  function __fzf_complete_native
    set -l -- token (commandline -t)
    set -l -- completions (eval complete -C \"$argv[1]\")
    test -n "$completions"; or begin commandline -f repaint; return; end

    # Calculate tabstop based on longest completion item (sample first 500 for performance)
    set -l -- tabstop 20
    set -l -- sample_size (math "min(500, "(count $completions)")")
    for c in $completions[1..$sample_size]
      set -l -- len (string length -V -- (string split -- \t $c))
      test -n "$len[2]" -a "$len[1]" -gt "$tabstop"
      and set -- tabstop $len[1]
    end
    # limit to 120 to prevent long lines
    set -- tabstop (math "min($tabstop + 4, 120)")

    set -l result
    set -lx -- FZF_DEFAULT_OPTS (__fzf_defaults \
      "--reverse --delimiter=\\t --nth=1 --tabstop=$tabstop --color=fg:dim,nth:regular" \
      $FZF_COMPLETION_OPTS $argv[2..-1] --accept-nth=1 --read0 --print0)
    set -- result (string join0 -- $completions | eval (__fzfcmd) | string split0)
    and begin
      set -l -- tail ' '
      # Append / to bare ~username results (fish omits it unlike other shells)
      set -- result (string replace -r -- '^(~\w+)\s?$' '$1/' $result)
      # Don't add trailing space if single result is a directory
      test (count $result) -eq 1
      and string match -q -- '*/' "$result"; and set -- tail ''

      set -l -- result (string escape -n -- $result)

      string match -q -- '~*' "$token"
      and set result (string replace -r -- '^\\\\~' '~' $result)

      string match -q -- '$*' "$token"
      and set result (string replace -r -- '^\\\\\$' '\$' $result)

      commandline -rt -- (string join ' ' -- $result)$tail
    end
    commandline -f repaint
  end

  function _fzf_complete
    set -l -- args (string escape -- $argv | string join ' ' | string split -- ' -- ')
    set -l -- post_func (status function)_(string split -- ' ' $args[2])[1]_post
    set -lx -- FZF_DEFAULT_OPTS (__fzf_defaults --reverse $FZF_COMPLETION_OPTS $args[1])
    set -lx FZF_DEFAULT_OPTS_FILE
    set -lx FZF_DEFAULT_COMMAND
    set -l -- fzf_query (commandline -t | string escape)
    set -l result
    eval (__fzfcmd) --query=$fzf_query | while read -l r; set -a -- result $r; end
    and if functions -q $post_func
      commandline -rt -- (string collect -- $result | eval $post_func $args[2] | string join ' ')' '
    else
      commandline -rt -- (string join -- ' ' (string escape -- $result))' '
    end
    commandline -f repaint
  end

  # Kill completion (process selection)
  function _fzf_complete_kill
    set -l -- fzf_query (commandline -t | string escape)
    set -lx -- FZF_DEFAULT_OPTS (__fzf_defaults --reverse $FZF_COMPLETION_OPTS \
    --accept-nth=2 -m --header-lines=1 --no-preview --wrap)
    set -lx FZF_DEFAULT_OPTS_FILE
    if type -q ps
      set -l -- ps_cmd 'begin command ps -eo user,pid,ppid,start,time,command 2>/dev/null;' \
      'or command ps -eo user,pid,ppid,time,args 2>/dev/null;' \
      'or command ps --everyone --full --windows 2>/dev/null; end'
      set -l -- result (eval $ps_cmd \| (__fzfcmd) --query=$fzf_query)
      and commandline -rt -- (string join ' ' -- $result)" "
    else
      __fzf_complete_native "kill " --multi --query=$fzf_query
    end
    commandline -f repaint
  end

  # Main completion function
  function fzf-completion
    set -l -- tokens (__fzf_cmd_tokens)
    set -l -- current_token (commandline -t)
    set -l -- cmd_name $tokens[1]

    # Route to appropriate completion function
    if test -n "$tokens"; and functions -q _fzf_complete_$cmd_name
      _fzf_complete_$cmd_name $tokens
    else
      set -l -- fzf_opt --query=$current_token --multi
      __fzf_complete_native "$tokens $current_token" $fzf_opt
    end
  end

  # Bind Shift-Tab to fzf-completion (Tab retains native Fish behavior)
  if test (string match -r -- '^\d+' $version) -ge 4
    bind shift-tab fzf-completion
    bind -M insert shift-tab fzf-completion
  else
    bind -k btab fzf-completion
    bind -M insert -k btab fzf-completion
  end
end

# Run setup
fzf_completion_setup
