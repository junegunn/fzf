#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ completion.fish
#
# - $FZF_COMPLETION_TRIGGER               (default: '**')
# - $FZF_COMPLETION_OPTS                  (default: empty)
# - $FZF_COMPLETION_PATH_OPTS             (default: empty)
# - $FZF_COMPLETION_DIR_OPTS              (default: empty)
# - $FZF_COMPLETION_FILE_OPTS             (default: empty)
# - $FZF_COMPLETION_DIR_COMMANDS          (default: see variable declaration for default values)
# - $FZF_COMPLETION_FILE_COMMANDS         (default: see variable declaration for default values)
# - $FZF_COMPLETION_NATIVE_COMMANDS       (default: see variable declaration for default values)
# - $FZF_COMPLETION_NATIVE_COMMANDS_MULTI (default: see variable declaration for default values)
# - $FZF_COMPLETION_SUBCOMMAND_COMMANDS   (default: see variable declaration for default values)
# - $FZF_COMPLETION_OVERRIDE_TAB          (default: empty, set to 1 to use fzf for all tab completions)

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
    set -l result
    if type -q column
      set -lx -- FZF_DEFAULT_OPTS (__fzf_defaults --reverse \
      $FZF_COMPLETION_OPTS $argv[2..-1] --accept-nth=1)
      set -- result (eval complete -C \"$argv[1]\" \| column -t -s \\t \| (__fzfcmd))
    else
      set -lx -- FZF_DEFAULT_OPTS (__fzf_defaults "--reverse --nth=1 --color=fg:dim,nth:regular" \
      $FZF_COMPLETION_OPTS $argv[2..-1] --accept-nth=1)
      set -- result (eval complete -C \"$argv[1]\" \| (__fzfcmd))
    end
    and commandline -rt -- (string join ' ' -- $result)' '
    commandline -f repaint
  end

  # Generic path completion
  function __fzf_generic_path_completion
    set -lx -- dir $argv[1]
    set -l -- fzf_query $argv[2]
    set -l -- opt_prefix $argv[3]
    set -l -- compgen $argv[4]
    set -l -- tail " "

    # Set fzf options
    set -lx -- FZF_DEFAULT_OPTS (__fzf_defaults "--reverse --scheme=path" $FZF_COMPLETION_OPTS --print0)
    set -lx FZF_DEFAULT_COMMAND
    set -lx FZF_DEFAULT_OPTS_FILE

    if string match -q -- '*dir*' $compgen
      set -- tail ""
      set -a -- FZF_DEFAULT_OPTS --walker=dir,follow $FZF_COMPLETION_DIR_OPTS
    else if string match -q -- '*file*' $compgen
      set -a -- FZF_DEFAULT_OPTS --multi --walker=file,follow,hidden $FZF_COMPLETION_FILE_OPTS
    else
      set -a -- FZF_DEFAULT_OPTS --multi --walker=file,dir,follow,hidden $FZF_COMPLETION_PATH_OPTS
    end

    # Run fzf
    set -l result
    if functions -q "$compgen"
      set -- result (eval $compgen $dir \| (__fzfcmd) --query=$fzf_query | string split0)
    else
      set -- result (eval (__fzfcmd) --walker-root=$dir --query=$fzf_query | string split0)
    end
    and commandline -rt -- (string join -- ' ' $opt_prefix(string escape -n -- $result))$tail

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
    set -q FZF_COMPLETION_TRIGGER
    or set -l -- FZF_COMPLETION_TRIGGER '**'

    # Set variables containing the major and minor fish version numbers, using
    # a method compatible with all supported fish versions.
    set -l -- fish_major (string match -r -- '^\d+' $version)
    set -l -- fish_minor (string match -r -- '^\d+\.(\d+)' $version)[2]

    # Get tokens - use version-appropriate flags
    set -l tokens
    if test $fish_major -ge 4
      set -- tokens (commandline -xpc)
    else
      set -- tokens (commandline -opc)
    end

    # Filter out leading environment variable assignments
    set -l -- var_count 0
    for i in $tokens
      if string match -qr -- '^[\w]+=' $i
        set var_count (math $var_count + 1)
      else
        break
      end
    end
    set -e -- tokens[0..$var_count]

    set -l -- current_token (commandline -t)
    set -l -- cmd_name $tokens[1]

    set -l -- regex_trigger (string escape --style=regex -- $FZF_COMPLETION_TRIGGER)'$'
    set -l -- has_trigger false
    string match -qr -- $regex_trigger $current_token
    and set has_trigger true

    # Strip trigger from commandline before parsing
    if $has_trigger; and test -n "$FZF_COMPLETION_TRIGGER" -a -n "$current_token"
      set -- current_token (string replace -r -- $regex_trigger '' $current_token)
      commandline -rt -- $current_token
    end

    set -l -- parsed (__fzf_parse_commandline)
    set -l -- dir $parsed[1]
    set -l -- fzf_query $parsed[2]
    set -l -- opt_prefix $parsed[3]
    set -l -- full_query $opt_prefix$fzf_query

    if not $has_trigger
      if test -n "$FZF_COMPLETION_OVERRIDE_TAB"
        # Native completion commands (multi-selection)
        set -q FZF_COMPLETION_NATIVE_COMMANDS_MULTI
        or set -l -- FZF_COMPLETION_NATIVE_COMMANDS_MULTI set functions type
        set -l -- fzf_opt --select-1 --query=$full_query
        contains -- "$cmd_name" $FZF_COMPLETION_NATIVE_COMMANDS_MULTI
        and set -a -- fzf_opt --multi
        __fzf_complete_native "$tokens $current_token" $fzf_opt
      else
        commandline -f complete
      end
      return
    else if test -z "$tokens"
      __fzf_complete_native "" --query=$full_query
      return
    end

    set -l -- disable_opt_comp false
    if not test "$fish_major" -eq 3 -a "$fish_minor" -lt 3
      string match -qe -- ' -- ' (string sub -l (commandline -Cp) -- (commandline -p))
      and set -- disable_opt_comp true
    end

    if not $disable_opt_comp
      # Not using the --groups-only option of string-match, because it is
      # not available on fish versions older that 3.4.0
      set -l -- cmd_opt (string match -r -- '^(-{1,2})([\w.,:+-]*)$' $current_token)
      if test -n "$cmd_opt[2]" -a \( "$cmd_opt[2]" = -- -o (string length -- "$cmd_opt[3]") -ne 1 \)
        __fzf_complete_native "$tokens $cmd_opt[2]" --query=$cmd_opt[3] --multi
        return
      end
    end

    # Directory commands
    set -q FZF_COMPLETION_DIR_COMMANDS
    or set -l -- FZF_COMPLETION_DIR_COMMANDS cd pushd rmdir

    # File-only commands
    set -q FZF_COMPLETION_FILE_COMMANDS
    or set -l -- FZF_COMPLETION_FILE_COMMANDS cat head tail less more nano sed sort uniq wc patch source \
    bunzip2 bzip2 gunzip gzip

    # Native completion commands
    set -q FZF_COMPLETION_NATIVE_COMMANDS
    or set -l -- FZF_COMPLETION_NATIVE_COMMANDS ftp hg sftp ssh svn telnet

    # Native completion commands (multi-selection)
    set -q FZF_COMPLETION_NATIVE_COMMANDS_MULTI
    or set -l -- FZF_COMPLETION_NATIVE_COMMANDS_MULTI set functions type

    # Subcommand programs (use native completion for first parameter only)
    set -q FZF_COMPLETION_SUBCOMMAND_COMMANDS
    or set -l -- FZF_COMPLETION_SUBCOMMAND_COMMANDS git docker kubectl cargo npm

    # Route to appropriate completion function
    if functions -q _fzf_complete_$cmd_name
      _fzf_complete_$cmd_name $tokens
    else if contains -- "$cmd_name" $FZF_COMPLETION_SUBCOMMAND_COMMANDS; and test (count $tokens) -eq 1
      __fzf_complete_native "$cmd_name " --query=$full_query
    else if contains -- "$cmd_name" $FZF_COMPLETION_NATIVE_COMMANDS $FZF_COMPLETION_NATIVE_COMMANDS_MULTI
      set -l -- fzf_opt --query=$full_query
      contains -- "$cmd_name" $FZF_COMPLETION_NATIVE_COMMANDS_MULTI
      and set -a -- fzf_opt --multi
      __fzf_complete_native "$tokens " $fzf_opt
    else if contains -- "$cmd_name" $FZF_COMPLETION_DIR_COMMANDS
      __fzf_generic_path_completion "$dir" "$fzf_query" "$opt_prefix" _fzf_compgen_dir
    else if contains -- "$cmd_name" $FZF_COMPLETION_FILE_COMMANDS
      __fzf_generic_path_completion "$dir" "$fzf_query" "$opt_prefix" _fzf_compgen_file
    else
      __fzf_generic_path_completion "$dir" "$fzf_query" "$opt_prefix" _fzf_compgen_path
    end
  end

  # Bind tab to fzf-completion
  bind \t fzf-completion
  bind -M insert \t fzf-completion
end

# Run setup
fzf_completion_setup
