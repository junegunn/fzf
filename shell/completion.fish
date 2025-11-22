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
                test -n "$FZF_TMUX" -a "$FZF_TMUX" != 0; or test -n "$FZF_TMUX_OPTS"
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

        # Extract option prefix if present (handles --option=value pattern)
        set -l opt_prefix ""
        if string match -q '*=*' -- "$base"
            set opt_prefix (string match -r '^[^=]*=' -- "$base")
            set base (string replace -r '^[^=]*=' '' -- "$base")
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
                        set result "$result$opt_prefix"(string escape -- "$item")" "
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

        # Extract the current token
        set -l current_token ""
        set -l token_start 1
        set -l lbuf_len (string length -- "$lbuf")
        for i in (seq $lbuf_len -1 1)
            set -l char (string sub -s $i -l 1 -- "$lbuf")
            if test "$char" = ' ' -o "$char" = \t
                set token_start (math $i + 1)
                break
            end
        end
        set current_token (string sub -s $token_start -- "$lbuf")

        # Calculate prefix
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
        set -l new_len (math $token_start - 1)
        if test $new_len -gt 0
            set lbuf (string sub -l $new_len -- "$lbuf")
        else
            set lbuf ""
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
