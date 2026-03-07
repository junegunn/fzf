#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ key-bindings.fish
#
# - $FZF_CTRL_T_COMMAND
# - $FZF_CTRL_T_OPTS
# - $FZF_CTRL_R_COMMAND
# - $FZF_CTRL_R_OPTS
# - $FZF_ALT_C_COMMAND
# - $FZF_ALT_C_OPTS


# Key bindings
# ------------
# The oldest supported fish version is 3.1b1. To maintain compatibility, the
# command substitution syntax $(cmd) should never be used, even behind a version
# check, otherwise the source command will fail on fish versions older than 3.4.0.
function fzf_key_bindings

  # Check fish version
  if set -l -- fish_ver (string match -r '^(\d+)\.(\d+)' $version 2>/dev/null)
  and test "$fish_ver[2]" -lt 3 -o "$fish_ver[2]" -eq 3 -a "$fish_ver[3]" -lt 1
    echo "This script requires fish version 3.1b1 or newer." >&2
    return 1
  else if not type -q fzf
    echo "fzf was not found in path." >&2
    return 1
  end

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

  # Store current token in $dir as root for the 'find' command
  function fzf-file-widget -d "List files and folders"
    set -l commandline (__fzf_parse_commandline)
    set -lx dir $commandline[1]
    set -l fzf_query $commandline[2]
    set -l prefix $commandline[3]

    set -lx FZF_DEFAULT_OPTS (__fzf_defaults \
      "--reverse --walker=file,dir,follow,hidden --scheme=path" \
      "--multi $FZF_CTRL_T_OPTS --print0")

    set -lx FZF_DEFAULT_COMMAND "$FZF_CTRL_T_COMMAND"
    set -lx FZF_DEFAULT_OPTS_FILE

    set -l result (eval (__fzfcmd) --walker-root=$dir --query=$fzf_query | string split0)
    and commandline -rt -- (string join -- ' ' $prefix(string escape --no-quoted -- $result))' '

    commandline -f repaint
  end

  function fzf-history-widget -d "Show command history"
    set -l -- command_line (commandline)
    set -l -- current_line (commandline -L)
    set -l -- total_lines (count $command_line)
    set -l -- fzf_query (string escape -- $command_line[$current_line])

    set -lx -- FZF_DEFAULT_OPTS (__fzf_defaults '' \
      '--with-nth=2.. --nth=2..,.. --scheme=history --multi --no-multi-line' \
      '--no-wrap --wrap-sign="\t\t\t↳ " --preview-wrap-sign="↳ " --freeze-left=1' \
      '--bind="alt-enter:become(set -g fzf_temp {+sf3..}; string join0 -- (string split0 -- <$fzf_temp | fish_indent -i); unlink $fzf_temp &>/dev/null)"' \
      '--bind="alt-t:change-with-nth(1,3..|3..|2..)"' \
      '--bind="shift-delete:execute-silent(eval builtin history delete -Ce -- (string escape -n -- (string split0 -- <{+sf3..})))+reload(eval $FZF_DEFAULT_COMMAND)"' \
      "--bind=ctrl-r:toggle-sort,alt-r:toggle-raw --highlight-line $FZF_CTRL_R_OPTS" \
      '--accept-nth=3.. --delimiter="\t" --tabstop=4 --read0 --print0 --with-shell='(status fish-path)\\ -c)

    # Add dynamic preview options if preview command isn't already set by user
    if string match -qvr -- '--preview[= ]' "$FZF_DEFAULT_OPTS"
      # Prepend the options to allow user overrides
      set -p -- FZF_DEFAULT_OPTS \
        '--bind="focus,multi,resize:bg-transform:if test \\"$FZF_COLUMNS\\" -gt 100 -a \\\\( \\"$FZF_SELECT_COUNT\\" -gt 0 -o \\\\( -z \\"$FZF_WRAP\\" -a (string join0 -- <{f3..} | string length) -gt (math $FZF_COLUMNS - (switch $FZF_WITH_NTH; case 2..; echo 13; case 1,3..; echo 25; case 3..; echo 1; end)) \\\\) -o (string split0 -- <{sf3..} | fish_indent | count) -gt 1 \\\\); echo show-preview; else echo hide-preview; end"' \
        '--preview="test \\"$FZF_SELECT_COUNT\\" -gt 0; and string split0 -- <{+sf3..} | fish_indent --only-indent --ansi; and echo -n \\\\n; string collect -- \\\\#\\\\ {1} (string split0 -- <{sf3..}) | fish_indent --ansi"' \
        '--preview-window="right,50%,wrap-word,follow,info,hidden"'
    end

    set -lx FZF_DEFAULT_OPTS_FILE

    set -lx -- FZF_DEFAULT_COMMAND 'builtin history -z'

    # Enable syntax highlighting colors on fish v4.3.3 and newer
    if set -l -- v (string match -r -- '^(\d+)\.(\d+)(?:\.(\d+))?' $version)
    and test "$v[2]" -gt 4 -o "$v[2]" -eq 4 -a \
      \( "$v[3]" -gt 3 -o "$v[3]" -eq 3 -a \
      \( -n "$v[4]" -a "$v[4]" -ge 3 \) \)

      set -a -- FZF_DEFAULT_OPTS '--ansi'
      set -a -- FZF_DEFAULT_COMMAND '--color=always --show-time=(set_color $fish_color_comment)"%F %a %T%t%s%t"(set_color $fish_color_normal)'
    else
      set -a -- FZF_DEFAULT_COMMAND '--show-time="%F %a %T%t%s%t"'
    end

    # Merge history from other sessions before searching
    test -z "$fish_private_mode"; and builtin history merge

    if set -l result (eval $FZF_DEFAULT_COMMAND \| (__fzfcmd) --query=$fzf_query | string split0)
      if test "$total_lines" -eq 1
        commandline -- $result
      else
        set -l a (math $current_line - 1)
        set -l b (math $current_line + 1)
        commandline -- $command_line[1..$a] $result
        commandline -a -- '' $command_line[$b..-1]
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

  if not set -q FZF_CTRL_R_COMMAND; or test -n "$FZF_CTRL_R_COMMAND"
    if test -n "$FZF_CTRL_R_COMMAND"
      echo "warning: FZF_CTRL_R_COMMAND is set to a custom command, but custom commands are not yet supported for CTRL-R" >&2
    end
    bind \cr fzf-history-widget
    bind -M insert \cr fzf-history-widget
  end

  if not set -q FZF_CTRL_T_COMMAND; or test -n "$FZF_CTRL_T_COMMAND"
    bind \ct fzf-file-widget
    bind -M insert \ct fzf-file-widget
  end

  if not set -q FZF_ALT_C_COMMAND; or test -n "$FZF_ALT_C_COMMAND"
    bind \ec fzf-cd-widget
    bind -M insert \ec fzf-cd-widget
  end

end

# Run setup
fzf_key_bindings
