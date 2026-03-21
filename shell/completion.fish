#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ completion.fish
#
# - $FZF_COMPLETION_OPTS
# - $FZF_EXPANSION_OPTS

# The oldest supported fish version is 3.4.0. For this message being able to be
# displayed on older versions, the command substitution syntax $() should not
# be used anywhere in the script, otherwise the source command will fail.
if string match -qr -- '^[12]\\.|^3\\.[0-3]' $version
  echo "fzf completion script requires fish version 3.4.0 or newer." >&2
  return 1
else if not command -q fzf
  echo "fzf was not found in path." >&2
  return 1
end

function fzf_complete -w fzf -d 'fzf command completion and wildcard expansion search'
  # Restore the default shift-tab behavior on tab completions
  if commandline --paging-mode
    commandline -f complete-and-search
    return
  end

  # Remove any trailing unescaped backslash from token and update command line
  set -l -- token (string replace -r -- '(?<!\\\\)(?:\\\\\\\\)*\\K\\\\$' '' (commandline -t | string collect) | string collect)
  commandline -rt -- $token

  # Remove any line breaks from token
  set -- token (string replace -ra -- '\\\\\\n' '' $token | string collect)

  # regex: Match token with unescaped/unquoted glob character
  set -l -- r_glob '^(?:[^\'"\\\\*]|\\\\[\\S\\s]|\'(?:\\\\[\\S\\s]|[^\'\\\\])*\'|"(?:\\\\[\\S\\s]|[^"\\\\])*")*\\*[\\S\\s]*$'

  # regex: Match any unbalanced quote character
  set -l -- r_quote '^(?>(?:\\\\[\\s\\S]|"(?:[^"\\\\]|\\\\[\\s\\S])*"|\'(?:[^\'\\\\]|\\\\[\\s\\S])*\'|[^\'"\\\\]+)*)\\K[\'"]'

  # The expansion pattern is the token with any open quote closed, or is empty.
  set -l -- glob_pattern (string match -r -- $r_glob $token | string collect)(string match -r -- $r_quote $token | string collect -a)

  set -l -- cl_tokenize_opt '--tokens-expanded'
  string match -q -- '3.*' $version
  and set -- cl_tokenize_opt '--tokenize'

  # Set command line tokens without any leading variable definitions or launcher
  # commands (including their options, but not any option arguments).
  set -l -- r_cmd '^(?:(?:builtin|command|doas|env|sudo|\\w+=\\S*|-\\S+)\\s+)*\\K[\\s\\S]+'
  set -l -- cmd (commandline $cl_tokenize_opt --input=(commandline -pc | string match -r $r_cmd))
  test -z "$token"
  and set -a -- cmd ''

  # Set fzf options
  test -z "$FZF_TMUX_HEIGHT"
  and set -l -- FZF_TMUX_HEIGHT 40%

  set -lax -- FZF_DEFAULT_OPTS \
    "--height=$FZF_TMUX_HEIGHT --min-height=20+ --bind=ctrl-z:ignore" \
    (test -r "$FZF_DEFAULT_OPTS_FILE"; and string join -- ' ' <$FZF_DEFAULT_OPTS_FILE) \
    $FZF_DEFAULT_OPTS '--bind=alt-r:toggle-raw --multi --wrap=word --reverse' \
    (if test -n "$glob_pattern"; string collect -- $FZF_EXPANSION_OPTS; else;
      string collect -- $FZF_COMPLETION_OPTS; end; string escape -n -- $argv) \
    --with-shell=(status fish-path)\\ -c

  set -lx FZF_DEFAULT_OPTS_FILE

  set -l -- fzf_cmd fzf
  test "$FZF_TMUX" = 1
  and set -- fzf_cmd fzf-tmux $FZF_TMUX_OPTS -d$FZF_TMUX_HEIGHT --

  set -l result

  # Get the completion list from stdin when it's not a tty
  if not isatty stdin
    set -l -- custom_post_func _fzf_post_complete_$cmd[1]
    functions -q $custom_post_func
    or set -- custom_post_func _fzf_complete_$cmd[1]_post

    if functions -q $custom_post_func
      $fzf_cmd | $custom_post_func $cmd | while read -l r; set -a -- result $r; end
    else if string match -q -- '*--print0*' "$FZF_DEFAULT_OPTS"
      $fzf_cmd | while read -lz r; set -a -- result $r; end
    else
      $fzf_cmd | while read -l r; set -a -- result $r; end
    end

  # Wildcard expansion
  else if test -n "$glob_pattern"
    # Set the command to be run by fzf, so there is a visual indicator and an
    # easy way to abort on long recursive searches.
    set -lx -- FZF_DEFAULT_COMMAND "for i in $glob_pattern;" \
      'test -d "$i"; and string match -qv -- "*/" $i; and set -- i $i/;' \
      'string join0 -- $i; end'

    set -- result (string escape -n -- ($fzf_cmd --read0 --print0 --scheme=path --no-multi-line | string split0))

  # Command completion
  else
    # Call custom function if defined
    set -l -- custom_func _fzf_complete_$cmd[1]
    if functions -q $custom_func; and not set -q __fzf_no_custom_complete
      set -lx __fzf_no_custom_complete
      $custom_func $cmd
      return
    end

    # Workaround for complete not having newlines in results
    if string match -qr -- '\\n' $token
      set -- token (string replace -ra -- '(?<!\\\\)(?:\\\\\\\\)*\\K\\\\\$' '\\\\\\\\\$' $token | string collect)
      set -- token (string unescape -- $token | string collect)
      set -- token (string replace -ra -- '\\n' '\\\\n' $token | string collect)
    end

    set -- list (complete -C --escape -- (string join -- ' ' (commandline -pc $cl_tokenize_opt) $token | string collect))
    if test -n "$list"
      # Get the initial tabstop value
      if set -l -- tabstop (string match -rga -- '--tabstop[= ](?:0*)([1-9]\\d+|[4-9])' "$FZF_DEFAULT_OPTS")[-1]
        set -- tabstop (math $tabstop - 4)
      else
        set -- tabstop 4
      end

      # Determine the tabstop length for description alignment
      set -l -- max_columns (math $COLUMNS - 40)
      for i in $list[1..500]
        set -l -- item (string split -f 1 -- \t $i)
        and set -l -- len (string length -V -- $item)
        and test "$len" -gt "$tabstop" -a "$len" -lt "$max_columns"
        and set -- tabstop $len
      end
      set -- tabstop (math $tabstop + 4)

      set -- result (string collect -- $list | $fzf_cmd --delimiter="\t" --tabstop=$tabstop --wrap-sign=\t"↳ " --accept-nth=1)
    end
  end

  # Update command line
  if test -n "$result"
    # No extra space after single selection that ends with path separator
    set -l -- tail ' '
    test (count $result) -eq 1
    and string match -q -- '*/' "$result"
    and set -- tail ''

    commandline -rt -- (string join -- ' ' $result)$tail
  end

  commandline -f repaint
end

function _fzf_complete
  set -l fzf_args
  for i in $argv
    string match -q -- '--' $i; and break
    set -a -- fzf_args $i
  end
  fzf_complete $fzf_args
end

# Bind to shift-tab
if string match -qr -- '^\\d\\d+|^[4-9]' $version
  bind shift-tab fzf_complete
  bind -M insert shift-tab fzf_complete
else
  bind -k btab fzf_complete
  bind -M insert -k btab fzf_complete
end
