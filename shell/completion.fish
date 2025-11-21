#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ completion.fish
#
# - $FZF_TMUX                     (default: 0)
# - $FZF_TMUX_OPTS                (default: empty)
# - $FZF_COMPLETION_TRIGGER       (default: '**')
# - $FZF_COMPLETION_OPTS          (default: empty)
# - $FZF_COMPLETION_PATH_OPTS     (default: empty)
# - $FZF_COMPLETION_DIR_OPTS      (default: empty)
# - $FZF_COMPLETION_FILE_OPTS     (default: empty)
# - $FZF_COMPLETION_DIR_COMMANDS  (default: cd pushd rmdir)
# - $FZF_COMPLETION_FILE_COMMANDS (default: cat head tail less more nano)
# - $FZF_COMPLETION_PATH_WALKER   (default: 'file,dir,follow,hidden')
# - $FZF_COMPLETION_DIR_WALKER    (default: 'dir,follow')
# - $FZF_COMPLETION_FILE_WALKER   (default: 'file,follow,hidden')

function fzf_completion_setup
    # Check fish version
    set -l fish_ver (string match -r '^(\d+).(\d+)' $version 2> /dev/null; or echo 0\n0\n0)
    if test \( "$fish_ver[2]" -lt 3 \) -o \( "$fish_ver[2]" -eq 3 -a "$fish_ver[3]" -lt 1 \)
        echo "This script requires fish version 3.1b1 or newer." >&2
        return 1
    else if not type -q fzf
        echo "fzf was not found in path." >&2
        return 1
    end

    function __fzf_completion_defaults
        # $argv[1]: Prepend to FZF_DEFAULT_OPTS_FILE and FZF_DEFAULT_OPTS
        # $argv[2..]: Append to FZF_DEFAULT_OPTS_FILE and FZF_DEFAULT_OPTS
        test -n "$FZF_TMUX_HEIGHT"; or set -l FZF_TMUX_HEIGHT 40%
        string join ' ' -- \
            "--height $FZF_TMUX_HEIGHT --min-height=20+ --bind=ctrl-z:ignore" $argv[1] \
            (test -r "$FZF_DEFAULT_OPTS_FILE"; and string join -- ' ' <$FZF_DEFAULT_OPTS_FILE) \
            $FZF_DEFAULT_OPTS $argv[2..-1]
    end

    function __fzf_comprun
        # $argv[1]: command word (for custom _fzf_comprun function)
        # $argv[2..]: fzf arguments
        if type -q _fzf_comprun
            _fzf_comprun $argv
        else if test -n "$TMUX_PANE"; and begin
                test "$FZF_TMUX" != 0; or test -n "$FZF_TMUX_OPTS"
            end
            set -l tmux_opts
            if test -n "$FZF_TMUX_OPTS"
                set tmux_opts $FZF_TMUX_OPTS
            else
                set tmux_opts -d$FZF_TMUX_HEIGHT
            end
            eval fzf-tmux $tmux_opts -- $argv[2..-1]
        else
            fzf $argv[2..-1]
        end
    end

    # Parse hostnames from SSH config, known_hosts, and /etc/hosts
    # Allow user to override this function by defining it before sourcing
    if not type -q __fzf_list_hosts
        function __fzf_list_hosts
            # Parse SSH config files
            for config_file in /etc/ssh/ssh_config ~/.ssh/config ~/.ssh/config.d/*
                if test -r "$config_file"
                    while read -l line
                        set -l lower_line (string lower -- "$line")
                        if string match -q -r '^\s*host(name)?\s' -- "$lower_line"
                            # Remove the Host/Hostname prefix
                            set -l hosts (string replace -r '^\s*[Hh]ost(name)?\s+' '' -- "$line")
                            # Remove comments
                            set hosts (string replace -r '#.*' '' -- "$hosts")
                            # Split on whitespace and filter wildcards
                            for host in (string split ' ' -- "$hosts")
                                if test -n "$host"; and not string match -q -r '[*?%]' -- "$host"
                                    echo $host
                                end
                            end
                        end
                    end <"$config_file"
                end
            end

            # Parse ~/.ssh/known_hosts
            if test -r ~/.ssh/known_hosts
                while read -l line
                    # Get first field (before space)
                    set -l first_field (string split ' ' -- "$line")[1]
                    # Skip if empty or starts with @
                    if test -z "$first_field"; or string match -q '@*' -- "$first_field"
                        continue
                    end
                    # Remove brackets and port numbers, split on comma
                    set first_field (string replace -r -a '\[|\]|:\d+' '' -- "$first_field")
                    for host in (string split ',' -- "$first_field")
                        if test -n "$host"
                            echo $host
                        end
                    end
                end <~/.ssh/known_hosts
            end

            # Parse /etc/hosts
            if test -r /etc/hosts
                while read -l line
                    # Remove comments
                    set line (string replace -r '#.*' '' -- "$line")
                    # Skip empty lines
                    if test -z "$line"
                        continue
                    end
                    # Get all fields except first (IP address)
                    set -l fields (string split -n ' ' -- "$line")
                    for i in (seq 2 (count $fields))
                        if test -n "$fields[$i]"; and test "$fields[$i]" != "0.0.0.0"
                            echo $fields[$i]
                        end
                    end
                end </etc/hosts
            end
        end
    end

    # Generic path completion
    function __fzf_generic_path_completion
        set -l base $argv[1]
        set -l lbuf $argv[2]
        set -l compgen $argv[3]
        set -l fzf_opts $argv[4]
        set -l suffix $argv[5]
        set -l tail $argv[6]
        set -l cmd_word $argv[7]

        # Don't complete if contains special patterns
        if string match -q -r '\$\(|<\(|>\(|:=|`' -- "$base"
            return
        end

        # Expand the base path
        set -l expanded_base (eval echo $base 2>/dev/null)
        if test $status -ne 0
            return
        end
        set base $expanded_base

        # Find the directory portion
        set -l dir ""
        if string match -q '*/*' -- "$base"
            set dir "$base"
        end

        # Walk up to find existing directory
        while true
            if test -z "$dir"; or test -d "$dir"
                set -l leftover ""
                if test -n "$dir"
                    set leftover (string replace -- "$dir" '' "$base")
                    set leftover (string replace -r '^/' '' -- "$leftover")
                else
                    set leftover "$base"
                end

                if test -z "$dir"
                    set dir '.'
                end
                # Remove trailing slash unless root
                if test "$dir" != /
                    set dir (string replace -r '/$' '' -- "$dir")
                end

                # Run fzf
                set -lx FZF_DEFAULT_OPTS (__fzf_completion_defaults "--reverse --scheme=path" "$FZF_COMPLETION_OPTS")
                set -lx FZF_DEFAULT_COMMAND
                set -lx FZF_DEFAULT_OPTS_FILE

                set -l matches
                if type -q "$compgen"
                    set matches (eval $compgen (string escape -- "$dir") | __fzf_comprun "$cmd_word" $fzf_opts -q "$leftover")
                else
                    set -l walker
                    set -l rest
                    if string match -q '*dir*' -- "$compgen"
                        if test -n "$FZF_COMPLETION_DIR_WALKER"
                            set walker $FZF_COMPLETION_DIR_WALKER
                        else
                            set walker "dir,follow"
                        end
                        set rest $FZF_COMPLETION_DIR_OPTS
                    else if string match -q '*file*' -- "$compgen"
                        if test -n "$FZF_COMPLETION_FILE_WALKER"
                            set walker $FZF_COMPLETION_FILE_WALKER
                        else
                            set walker "file,follow,hidden"
                        end
                        set rest $FZF_COMPLETION_FILE_OPTS
                    else
                        if test -n "$FZF_COMPLETION_PATH_WALKER"
                            set walker $FZF_COMPLETION_PATH_WALKER
                        else
                            set walker "file,dir,follow,hidden"
                        end
                        set rest $FZF_COMPLETION_PATH_OPTS
                    end
                    # Build args list, filtering empty values
                    set -l fzf_args -q "$leftover" --walker "$walker" --walker-root="$dir"
                    test -n "$fzf_opts"; and set -a fzf_args $fzf_opts
                    test -n "$rest"; and eval set -a fzf_args $rest
                    set matches (__fzf_comprun "$cmd_word" $fzf_args < /dev/tty)
                end

                if test -n "$matches"
                    set -l result ""
                    for item in $matches
                        set item (string replace -r "$suffix\$" '' -- "$item")$suffix
                        set result "$result"(string escape -- "$item")" "
                    end
                    set result (string trim -r -- "$result")
                    commandline -r -- "$lbuf$result$tail"
                end
                commandline -f repaint
                break
            end
            set dir (dirname -- "$dir")
            set dir (string replace -r '([^/])$' '$1/' -- "$dir")
        end
    end

    function _fzf_path_completion
        __fzf_generic_path_completion $argv[1] $argv[2] _fzf_compgen_path -m "" " " $argv[3]
    end

    function _fzf_dir_completion
        __fzf_generic_path_completion $argv[1] $argv[2] _fzf_compgen_dir "" / "" $argv[3]
    end

    function _fzf_file_completion
        __fzf_generic_path_completion $argv[1] $argv[2] _fzf_compgen_file -m "" " " $argv[3]
    end

    # SSH completion
    function _fzf_complete_ssh
        set -l tokens (commandline -opc)
        set -l last_token ""
        if test (count $tokens) -gt 1
            set last_token $tokens[-1]
        end

        # Check if previous token is -i, -F, or -E (file arguments)
        switch "$last_token"
            case -i -F -E
                _fzf_path_completion $argv[1] $argv[2] $argv[3]
                return
        end

        # Otherwise complete hostnames
        set -l user ""
        if string match -q '*@*' -- $argv[1]
            set user (string replace -r '@.*' '@' -- $argv[1])
        end

        set -lx FZF_DEFAULT_OPTS (__fzf_completion_defaults "--reverse" "$FZF_COMPLETION_OPTS")
        set -lx FZF_DEFAULT_OPTS_FILE

        set -l result (__fzf_list_hosts | sort -u | while read -l host
            echo "$user$host"
        end | __fzf_comprun $argv[3] +m -q (string replace -r '^[^@]*@' '' -- $argv[1]))

        if test -n "$result"
            commandline -r -- "$argv[2]$result "
        end
        commandline -f repaint
    end

    # Telnet completion (hostnames only)
    function _fzf_complete_telnet
        set -lx FZF_DEFAULT_OPTS (__fzf_completion_defaults "--reverse" "$FZF_COMPLETION_OPTS")
        set -lx FZF_DEFAULT_OPTS_FILE

        set -l result (__fzf_list_hosts | sort -u | __fzf_comprun $argv[3] +m -q $argv[1])

        if test -n "$result"
            commandline -r -- "$argv[2]$result "
        end
        commandline -f repaint
    end

    # Kill completion (process selection)
    function _fzf_complete_kill
        set -lx FZF_DEFAULT_OPTS (__fzf_completion_defaults "--reverse" "$FZF_COMPLETION_OPTS")
        set -lx FZF_DEFAULT_OPTS_FILE

        # Try different ps formats, capture output directly
        set -l ps_output (command ps -eo user,pid,ppid,start,time,command 2>/dev/null)
        if test $status -ne 0 -o -z "$ps_output"
            set ps_output (command ps -eo user,pid,ppid,time,args 2>/dev/null)
        end
        if test $status -ne 0 -o -z "$ps_output"
            set ps_output (command ps --everyone --full --windows 2>/dev/null)
        end

        set -l result (printf '%s\n' $ps_output | __fzf_comprun $argv[3] -m --header-lines=1 --no-preview --wrap -q $argv[1])

        if test -n "$result"
            # Extract PIDs (second field)
            set -l pids
            for line in $result
                set -l fields (string split -n ' ' -- "$line")
                if test (count $fields) -ge 2
                    set -a pids $fields[2]
                end
            end
            if test (count $pids) -gt 0
                commandline -r -- "$argv[2]"(string join ' ' -- $pids)" "
            end
        end
        commandline -f repaint
    end

    # Variable completion for 'set -e' (erase variable)
    function _fzf_complete_set
        set -l tokens (commandline -opc)

        # Check if we're erasing variables (set -e)
        set -l is_erase 0
        for token in $tokens
            if string match -q -r '^-.*e' -- "$token"
                set is_erase 1
                break
            end
        end

        if test $is_erase -eq 1
            set -lx FZF_DEFAULT_OPTS (__fzf_completion_defaults "--reverse" "$FZF_COMPLETION_OPTS")
            set -lx FZF_DEFAULT_OPTS_FILE

            set -l result (set -n | __fzf_comprun $argv[3] -m -q $argv[1])

            if test -n "$result"
                commandline -r -- "$argv[2]"(string join ' ' -- $result)" "
            end
            commandline -f repaint
        else
            # Fall back to path completion for other set operations
            _fzf_path_completion $argv[1] $argv[2] $argv[3]
        end
    end

    # Function completion for 'functions -e' (erase function)
    function _fzf_complete_functions
        set -l tokens (commandline -opc)

        # Check if we're erasing functions (functions -e or --erase)
        set -l is_erase 0
        for token in $tokens
            if string match -q -r '^-.*e|^--erase' -- "$token"
                set is_erase 1
                break
            end
        end

        if test $is_erase -eq 1
            set -lx FZF_DEFAULT_OPTS (__fzf_completion_defaults "--reverse" "$FZF_COMPLETION_OPTS")
            set -lx FZF_DEFAULT_OPTS_FILE

            set -l result (functions -n | __fzf_comprun $argv[3] -m -q $argv[1])

            if test -n "$result"
                commandline -r -- "$argv[2]"(string join ' ' -- $result)" "
            end
            commandline -f repaint
        else
            # Fall back to path completion
            _fzf_path_completion $argv[1] $argv[2] $argv[3]
        end
    end

    # Main completion function
    function fzf-completion
        set -l trigger (test -n "$FZF_COMPLETION_TRIGGER"; and echo "$FZF_COMPLETION_TRIGGER"; or echo '**')
        set -l lbuf (commandline -c)
        set -l tokens (commandline -opc)

        if test (count $tokens) -lt 1
            # No tokens, fall back to default completion
            commandline -f complete
            return
        end

        # Handle empty trigger with space
        if test -z "$trigger"; and test (string sub -s -1 -- "$lbuf") = ' '
            set -a tokens ""
        end

        # Check if the trigger is present at the end
        set -l tail (string sub -s -(string length -- "$trigger") -- "$lbuf")
        if test "$tail" != "$trigger"
            # No trigger, fall back to default completion
            commandline -f complete
            return
        end

        # Get the command word
        set -l cmd_word $tokens[1]

        # Get current token without glob expansion
        set -l current_token (commandline -t)
        set -l prefix
        if test -z "$trigger"
            set prefix "$current_token"
        else
            set -l trigger_len (string length -- "$trigger")
            set -l token_len (string length -- "$current_token")
            set -l prefix_len (math $token_len - $trigger_len)
            if test $prefix_len -gt 0
                set prefix (string sub -l $prefix_len -- "$current_token")
            else
                set prefix ""
            end
        end

        # Calculate lbuf without current token
        if test -n "$current_token"
            set -l token_len (string length -- "$current_token")
            set -l lbuf_len (string length -- "$lbuf")
            set -l new_len (math "$lbuf_len" - "$token_len")
            if test -n "$new_len" -a "$new_len" -gt 0
                set lbuf (string sub -l "$new_len" -- "$lbuf")
            else
                set lbuf ""
            end
        end

        # Directory commands
        set -l d_cmds
        if test -n "$FZF_COMPLETION_DIR_COMMANDS"
            set d_cmds (string split ' ' -- "$FZF_COMPLETION_DIR_COMMANDS")
        else
            set d_cmds cd pushd rmdir
        end

        # File-only commands
        set -l f_cmds
        if test -n "$FZF_COMPLETION_FILE_COMMANDS"
            set f_cmds (string split ' ' -- "$FZF_COMPLETION_FILE_COMMANDS")
        else
            set f_cmds cat head tail less more
        end

        # Route to appropriate completion function
        if type -q "_fzf_complete_$cmd_word"
            eval "_fzf_complete_$cmd_word" (string escape -- "$prefix") (string escape -- "$lbuf") (string escape -- "$cmd_word")
        else if contains -- "$cmd_word" $d_cmds
            _fzf_dir_completion "$prefix" "$lbuf" "$cmd_word"
        else if contains -- "$cmd_word" $f_cmds
            _fzf_file_completion "$prefix" "$lbuf" "$cmd_word"
        else
            _fzf_path_completion "$prefix" "$lbuf" "$cmd_word"
        end
    end

    # Bind tab to fzf-completion
    bind \t fzf-completion
    bind -M insert \t fzf-completion
end

# Run setup
fzf_completion_setup
