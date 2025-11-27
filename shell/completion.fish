#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ completion.fish
#
# - $FZF_COMPLETION_TRIGGER         (default: '**')
# - $FZF_COMPLETION_OPTS            (default: empty)
# - $FZF_COMPLETION_PATH_OPTS       (default: empty)
# - $FZF_COMPLETION_DIR_OPTS        (default: empty)
# - $FZF_COMPLETION_FILE_OPTS       (default: empty)
# - $FZF_COMPLETION_DIR_COMMANDS    (default: cd pushd rmdir)
# - $FZF_COMPLETION_FILE_COMMANDS   (default: cat head tail less more nano)
# - $FZF_COMPLETION_NATIVE_COMMANDS (default: ssh telnet set functions type)

function fzf_completion_setup
    # Load helper functions
    fzf_key_bindings

    # Use complete builtin for specific commands
    function __fzf_complete_native
        # Have the command run in a subshell
        set -lx -- FZF_DEFAULT_COMMAND "complete -C \"$argv[1]\""

        set -lx -- FZF_DEFAULT_OPTS (__fzf_defaults '--reverse --nth=1 --color=fg:dim,nth:regular' \
            $FZF_COMPLETION_OPTS $argv[2..-1] '--accept-nth=1 --with-shell='(status fish-path)\\ -c)

        set -l result (eval (__fzfcmd))
        and commandline -rt -- $result

        commandline -f repaint
    end

    # Generic path completion
    function __fzf_generic_path_completion
        set -lx dir $argv[1]
        set -l fzf_query $argv[2]
        set -l opt_prefix $argv[3]
        set -l compgen $argv[4]
        set -l tail " "

        # Set fzf options
        set -lx -- FZF_DEFAULT_OPTS (__fzf_defaults "--reverse --scheme=path" "$FZF_COMPLETION_OPTS --print0")
        set -lx FZF_DEFAULT_COMMAND
        set -lx FZF_DEFAULT_OPTS_FILE

        if string match -q -- '*dir*' $compgen
            set tail ""
            set -a -- FZF_DEFAULT_OPTS "--walker=dir,follow $FZF_COMPLETION_DIR_OPTS"
        else if string match -q -- '*file*' $compgen
            set -a -- FZF_DEFAULT_OPTS "-m --walker=file,follow,hidden $FZF_COMPLETION_FILE_OPTS"
        else
            set -a -- FZF_DEFAULT_OPTS "-m --walker=file,dir,follow,hidden $FZF_COMPLETION_PATH_OPTS"
        end

        # Run fzf
        if functions -q "$compgen"
            set -l result (eval $compgen $dir | eval (__fzfcmd) --query=$fzf_query | string split0)
            and commandline -rt -- (string join -- ' ' $opt_prefix(string escape -n -- $result))$tail
        else
            set -l result (eval (__fzfcmd) --walker-root=$dir --query=$fzf_query | string split0)
            and commandline -rt -- (string join -- ' ' $opt_prefix(string escape -n -- $result))$tail
        end

        commandline -f repaint
    end

    # Kill completion (process selection)
    function _fzf_complete_kill
        set -lx FZF_DEFAULT_OPTS (__fzf_defaults "--reverse" "$FZF_COMPLETION_OPTS")
        set -lx FZF_DEFAULT_OPTS_FILE

        set -l result (begin
            command ps -eo user,pid,ppid,start,time,command 2>/dev/null
            or command ps -eo user,pid,ppid,time,args 2>/dev/null # BusyBox
            or command ps --everyone --full --windows 2>/dev/null # Cygwin
        end | eval (__fzfcmd) --accept-nth=2 -m --header-lines=1 --no-preview --wrap --query=$argv[1])
        and commandline -rt -- (string join ' ' -- $result)" "
        commandline -f repaint
    end

    # Main completion function
    function fzf-completion
        set -q FZF_COMPLETION_TRIGGER
        or set -l FZF_COMPLETION_TRIGGER '**'

        # Get tokens - use version-appropriate flags
        set -l tokens
        if test (string match -r -- '^\d+' $version) -ge 4
            set tokens (commandline -xpc)
        else
            set tokens (commandline -opc)
        end

        # Handle empty trigger with space
        if test -z "$FZF_COMPLETION_TRIGGER"; and test (string sub -s -1 -- (commandline -c)) = ' '
            set -a tokens ""
        end

        set -l current_token (commandline -t)

        # Get the command name
        set -l cmd_name $tokens[1]

        # Check if token ends with trigger and strip it
        set -l trigger_len (string length -- "$FZF_COMPLETION_TRIGGER")
        set -l has_trigger false
        if test -n "$current_token"; and test (string length -- "$current_token") -ge $trigger_len
            set -l token_suffix (string sub -s -$trigger_len -- "$current_token")
            if test "$token_suffix" = "$FZF_COMPLETION_TRIGGER"
                set has_trigger true
                commandline -rt -- (string sub -e -$trigger_len -- "$current_token")
                set current_token (commandline -t)
            end
        end

        # Parse commandline (now without trigger)
        set -l parsed (__fzf_parse_commandline)
        set -l dir $parsed[1]
        set -l fzf_query $parsed[2]
        set -l opt_prefix $parsed[3]

        # Check if completing a flag/option (but not option with value like -o/path)
        if string match -q -- '-*' "$current_token"; and test -z "$opt_prefix"; and $has_trigger
            set -l -- fzf_opt --query=(string escape -- "$current_token")
            set -l -- complete_cmd "$cmd_name $current_token"
            __fzf_complete_native "$complete_cmd" $fzf_opt
            return
        end

        # Directory commands
        set -q FZF_COMPLETION_DIR_COMMANDS
        or set -l FZF_COMPLETION_DIR_COMMANDS cd pushd rmdir

        # File-only commands
        set -q FZF_COMPLETION_FILE_COMMANDS
        or set -l FZF_COMPLETION_FILE_COMMANDS cat head tail less more

        # Native completion commands
        set -q FZF_COMPLETION_NATIVE_COMMANDS
        or set -l FZF_COMPLETION_NATIVE_COMMANDS ssh telnet set functions type

        # If no trigger, fall back to native completion
        if not $has_trigger
            commandline -f complete
            return
        end

        # Route to appropriate completion function
        if functions -q _fzf_complete_$cmd_name
            _fzf_complete_$cmd_name "$fzf_query" "$cmd_name"
        else if contains -- "$cmd_name" $FZF_COMPLETION_NATIVE_COMMANDS
            set -l -- fzf_opt --query=(commandline -t | string escape)
            __fzf_complete_native "$cmd_name " $fzf_opt
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
