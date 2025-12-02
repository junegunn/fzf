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
