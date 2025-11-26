#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ completion.fish
#
# - $FZF_COMPLETION_TRIGGER         (default: '**'
# - $FZF_COMPLETION_OPTS            (default: empty)
# - $FZF_COMPLETION_PATH_OPTS       (default: empty)
# - $FZF_COMPLETION_DIR_OPTS        (default: empty)
# - $FZF_COMPLETION_FILE_OPTS       (default: empty)
# - $FZF_COMPLETION_DIR_COMMANDS    (default: cd pushd rmdir)
# - $FZF_COMPLETION_FILE_COMMANDS   (default: cat head tail less more nano)
# - $FZF_COMPLETION_NATIVE_COMMANDS (default: ssh telnet set functions)
# - $FZF_COMPLETION_PATH_WALKER     (default: 'file,dir,follow,hidden')
# - $FZF_COMPLETION_DIR_WALKER      (default: 'dir,follow')
# - $FZF_COMPLETION_FILE_WALKER     (default: 'file,follow,hidden')
# - $FZF_COMPLETION_NATIVE_MODE     (default: 'complete', or 'complete-and-search')

function fzf_completion_setup
    # Load helper functions
    fzf_key_bindings

    # Check fish version
    set -l fish_ver (string match -r '^(\d+).(\d+)' $version 2> /dev/null; or echo 0\n0\n0)
    if test \( "$fish_ver[2]" -lt 3 \) -o \( "$fish_ver[2]" -eq 3 -a "$fish_ver[3]" -lt 1 \)
        echo "This script requires fish version 3.1b1 or newer." >&2
        return 1
    else if not type -q fzf
        echo "fzf was not found in path." >&2
        return 1
    end

    # Delegate to native fish completion for specific commands
    # Use FZF_COMPLETION_NATIVE_MODE to choose: 'complete' (default) or 'complete-and-search'
    function __fzf_complete_native
        if test "$FZF_COMPLETION_NATIVE_MODE" = complete-and-search
            commandline -f complete-and-search
        else
            commandline -f complete
        end
    end

    # Generic path completion
    function __fzf_generic_path_completion
        set -lx dir $argv[1]
        set -l fzf_query $argv[2]
        set -l opt_prefix $argv[3]
        set -l compgen $argv[4]
        set -l fzf_opts $argv[5]
        set -l tail $argv[6]

        # Determine walker based on compgen type
        set -l walker
        set -l rest
        if string match -q '*dir*' -- "$compgen"
            set walker (test -n "$FZF_COMPLETION_DIR_WALKER"; and echo $FZF_COMPLETION_DIR_WALKER; or echo "dir,follow")
            set rest $FZF_COMPLETION_DIR_OPTS
        else if string match -q '*file*' -- "$compgen"
            set walker (test -n "$FZF_COMPLETION_FILE_WALKER"; and echo $FZF_COMPLETION_FILE_WALKER; or echo "file,follow,hidden")
            set rest $FZF_COMPLETION_FILE_OPTS
        else
            set walker (test -n "$FZF_COMPLETION_PATH_WALKER"; and echo $FZF_COMPLETION_PATH_WALKER; or echo "file,dir,follow,hidden")
            set rest $FZF_COMPLETION_PATH_OPTS
        end

        # Set fzf options
        set -lx FZF_DEFAULT_OPTS (__fzf_defaults \
            "--reverse --walker=$walker --scheme=path" \
            "$FZF_COMPLETION_OPTS $fzf_opts --print0 $rest")
        set -lx FZF_DEFAULT_COMMAND
        set -lx FZF_DEFAULT_OPTS_FILE

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

    function _fzf_path_completion
        __fzf_generic_path_completion $argv[1] $argv[2] $argv[3] _fzf_compgen_path -m " "
    end

    function _fzf_dir_completion
        __fzf_generic_path_completion $argv[1] $argv[2] $argv[3] _fzf_compgen_dir "" ""
    end

    function _fzf_file_completion
        __fzf_generic_path_completion $argv[1] $argv[2] $argv[3] _fzf_compgen_file -m " "
    end

    # Kill completion (process selection)
    function _fzf_complete_kill
        set -lx FZF_DEFAULT_OPTS (__fzf_defaults "--reverse" "$FZF_COMPLETION_OPTS")
        set -lx FZF_DEFAULT_OPTS_FILE

        set -l result (begin
            command ps -eo user,pid,ppid,start,time,command 2>/dev/null
            or command ps -eo user,pid,ppid,time,args 2>/dev/null # BusyBox
            or command ps --everyone --full --windows 2>/dev/null # Cygwin
        end | eval (__fzfcmd) -m --header-lines=1 --no-preview --wrap --print0 --query=$argv[1] | string split0)

        if test (count $result) -gt 0
            set -l pids
            for line in $result
                test -z "$line"; and continue
                set -l fields (string split -n ' ' -- "$line")
                if test (count $fields) -ge 2
                    set -a pids $fields[2]
                end
            end
            if test (count $pids) -gt 0
                commandline -rt -- (string join ' ' -- $pids)" "
            end
        end
        commandline -f repaint
    end

    # Main completion function
    function fzf-completion
        set -l trigger (test -n "$FZF_COMPLETION_TRIGGER"; and echo "$FZF_COMPLETION_TRIGGER"; or echo '**')

        # Get tokens - use version-appropriate flags
        # Fish 4.0+: -x (--tokens-expanded) returns expanded tokens
        # Fish 3.1-3.7: -o (--tokenize) returns tokenized output
        set -l fish_major (string match -r -- '^\d+' $version)

        set -l tokens
        if test "$fish_major" -ge 4
            set tokens (commandline -xpc)
        else
            set tokens (commandline -opc)
        end

        if test (count $tokens) -lt 1
            __fzf_complete_native
            return
        end

        # Handle empty trigger with space
        if test -z "$trigger"; and test (string sub -s -1 -- (commandline -c)) = ' '
            set -a tokens ""
        end

        # Check if the trigger is present at the end
        if test (string sub -s -(string length -- "$trigger") -- (commandline -c)) != "$trigger"
            __fzf_complete_native
            return
        end

        # Get the command word
        set -l cmd_word $tokens[1]

        # Strip trigger from commandline before parsing
        set -l raw_token (commandline --current-token 2>/dev/null | string collect)
        if test -n "$trigger"
            set -l stripped_token (string replace -r (string escape --style=regex -- "$trigger")'$' '' -- "$raw_token")
            commandline -rt -- "$stripped_token"
        end

        # Parse commandline (now without trigger)
        set -l parsed (__fzf_parse_commandline)
        set -l dir $parsed[1]
        set -l fzf_query $parsed[2]
        set -l opt_prefix $parsed[3]

        # Directory commands
        set -l d_cmds (string split ' ' -- "$FZF_COMPLETION_DIR_COMMANDS")
        or set d_cmds cd pushd rmdir

        # File-only commands
        set -l f_cmds (string split ' ' -- "$FZF_COMPLETION_FILE_COMMANDS")
        or set f_cmds cat head tail less more

        # Native completion commands
        set -l n_cmds (string split ' ' -- "$FZF_COMPLETION_NATIVE_COMMANDS")
        or set n_cmds ssh telnet set functions

        # Route to appropriate completion function
        if functions -q "_fzf_complete_$cmd_word"
            eval "_fzf_complete_$cmd_word" (string escape -- "$fzf_query") (string escape -- "$cmd_word")
        else if contains -- "$cmd_word" $n_cmds
            __fzf_complete_native
        else if contains -- "$cmd_word" $d_cmds
            _fzf_dir_completion "$dir" "$fzf_query" "$opt_prefix"
        else if contains -- "$cmd_word" $f_cmds
            _fzf_file_completion "$dir" "$fzf_query" "$opt_prefix"
        else
            _fzf_path_completion "$dir" "$fzf_query" "$opt_prefix"
        end
    end

    # Bind tab to fzf-completion
    bind \t fzf-completion
    bind -M insert \t fzf-completion
end

# Run setup
fzf_completion_setup
