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
# - $FZF_COMPLETION_DIR_COMMANDS          (default: cd pushd rmdir)
# - $FZF_COMPLETION_FILE_COMMANDS         (default: cat head tail less more nano)
# - $FZF_COMPLETION_NATIVE_COMMANDS       (default: ssh telnet)
# - $FZF_COMPLETION_NATIVE_COMMANDS_MULTI (default: set functions type)

function fzf_completion_setup
    # Load helper functions
    fzf_key_bindings

    # Use complete builtin for specific commands
    function __fzf_complete_native
        set -l result
        if type -q column
            set -lx -- FZF_DEFAULT_OPTS (__fzf_defaults --reverse \
                $FZF_COMPLETION_OPTS $argv[2..-1] --accept-nth=1)
            set result (eval complete -C \"$argv[1]\" \| column -t -s \\t \| (__fzfcmd))
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
            set -a -- FZF_DEFAULT_OPTS "--walker=dir,follow $FZF_COMPLETION_DIR_OPTS"
        else if string match -q -- '*file*' $compgen
            set -a -- FZF_DEFAULT_OPTS "-m --walker=file,follow,hidden $FZF_COMPLETION_FILE_OPTS"
        else
            set -a -- FZF_DEFAULT_OPTS "-m --walker=file,dir,follow,hidden $FZF_COMPLETION_PATH_OPTS"
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

    # Kill completion (process selection)
    function _fzf_complete_kill
        set -lx -- FZF_DEFAULT_OPTS (__fzf_defaults --reverse $FZF_COMPLETION_OPTS \
            --accept-nth=2 -m --header-lines=1 --no-preview --wrap)
        set -lx FZF_DEFAULT_OPTS_FILE
        if type -q ps
            set -l -- ps_cmd 'begin command ps -eo user,pid,ppid,start,time,command 2>/dev/null;' \
                'or command ps -eo user,pid,ppid,time,args 2>/dev/null;' \
                'or command ps --everyone --full --windows 2>/dev/null; end'
            set -l -- result (eval $ps_cmd \| (__fzfcmd) --query=$argv[1])
            and commandline -rt -- (string join ' ' -- $result)" "
        else
            __fzf_complete_native "kill " --multi --query=$argv[1]
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

        if not $has_trigger
            commandline -f complete
            return
        else if test -z "$tokens"
            __fzf_complete_native "" --query=$opt_prefix$fzf_query
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
                __fzf_complete_native "$cmd_name $cmd_opt[2]" --query=$cmd_opt[3] --multi
                return
            end
        end

        # Directory commands
        set -q FZF_COMPLETION_DIR_COMMANDS
        or set -l -- FZF_COMPLETION_DIR_COMMANDS cd pushd rmdir

        # File-only commands
        set -q FZF_COMPLETION_FILE_COMMANDS
        or set -l -- FZF_COMPLETION_FILE_COMMANDS cat head tail less more nano

        # Native completion commands
        set -q FZF_COMPLETION_NATIVE_COMMANDS
        or set -l -- FZF_COMPLETION_NATIVE_COMMANDS ssh telnet

        # Native completion commands (multi-selection)
        set -q FZF_COMPLETION_NATIVE_COMMANDS_MULTI
        or set -l -- FZF_COMPLETION_NATIVE_COMMANDS_MULTI set functions type

        # Route to appropriate completion function
        if contains -- "$cmd_name" $FZF_COMPLETION_NATIVE_COMMANDS $FZF_COMPLETION_NATIVE_COMMANDS_MULTI
            set -l -- fzf_opt --query=$fzf_query
            contains -- "$cmd_name" $FZF_COMPLETION_NATIVE_COMMANDS_MULTI
            and set -a -- fzf_opt --multi
            __fzf_complete_native "$cmd_name " $fzf_opt
        else if functions -q _fzf_complete_$cmd_name
            _fzf_complete_$cmd_name "$fzf_query" "$cmd_name"
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
