CHANGELOG
=========

0.25.0
------
- Text attributes set in `--color` are not reset when fzf sees another
  `--color` option for the same element. This allows you to put custom text
  attributes in your `$FZF_DEFAULT_OPTS` and still have those attributes
  even when you override the colors.

  ```sh
  # Default colors and attributes
  fzf

  # Apply custom text attributes
  export FZF_DEFAULT_OPTS='--color fg+:italic,hl:-1:underline,hl+:-1:reverse:underline'

  fzf

  # Different colors but you still have the attributes
  fzf --color hl:176,hl+:177

  # Write "regular" if you want to clear the attributes
  fzf --color hl:176:regular,hl+:177:regular
  ```
- Renamed `--phony` to `--disabled`
- You can dynamically enable and disable the search functionality using the
  new `enable-search`, `disable-search`, and `toggle-search` actions
- You can assign a different color to the query string for when search is disabled
  ```sh
  fzf --color query:#ffffff,disabled:#999999 --bind space:toggle-search
  ```
- Added `last` action to move the cursor to the last match
    - The opposite action `top` is renamed to `first`, but `top` is still
      recognized as a synonym for backward compatibility
- Added `preview-top` and `preview-bottom` actions
- Extended support for alt key chords: alt with any case-sensitive single character
  ```sh
  fzf --bind alt-,:first,alt-.:last
  ```

0.24.4
------
- Added `--preview-window` option `follow`
  ```sh
  # Preview window will automatically scroll to the bottom
  fzf --preview-window follow --preview 'for i in $(seq 100000); do
    echo "$i"
    sleep 0.01
    (( i % 300 == 0 )) && printf "\033[2J"
  done'
  ```
- Added `change-prompt` action
  ```sh
  fzf --prompt 'foo> ' --bind $'a:change-prompt:\x1b[31mbar> '
  ```
- Bug fixes and improvements

0.24.3
------
- Added `--padding` option
  ```sh
  fzf --margin 5% --padding 5% --border --preview 'cat {}' \
      --color bg:#222222,preview-bg:#333333
  ```

0.24.2
------
- Bug fixes and improvements

0.24.1
------
- Fixed broken `--color=[bw|no]` option

0.24.0
------
- Real-time rendering of preview window
  ```sh
  # fzf can render preview window before the command completes
  fzf --preview 'sleep 1; for i in $(seq 100); do echo $i; sleep 0.01; done'

  # Preview window can process ANSI escape sequence (CSI 2 J) for clearing the display
  fzf --preview 'for i in $(seq 100000); do
    (( i % 200 == 0 )) && printf "\033[2J"
    echo "$i"
    sleep 0.01
  done'
  ```
- Updated `--color` option to support text styles
  - `regular` / `bold` / `dim` / `underline` / `italic` / `reverse` / `blink`
    ```sh
    # * Set -1 to keep the original color
    # * Multiple style attributes can be combined
    # * Italic style may not be supported by some terminals
    rg --line-number --no-heading --color=always "" |
      fzf --ansi --prompt "Rg: " \
          --color fg+:italic,hl:underline:-1,hl+:italic:underline:reverse:-1 \
          --color pointer:reverse,prompt:reverse,input:159 \
          --pointer '  '
    ```
- More `--border` options
  - `vertical`, `top`, `bottom`, `left`, `right`
  - Updated Vim plugin to use these new `--border` options
    ```vim
    " Floating popup window in the center of the screen
    let g:fzf_layout = { 'window': { 'width': 0.9, 'height': 0.6 } }

    " Popup with 100% width
    let g:fzf_layout = { 'window': { 'width': 1.0, 'height': 0.5, 'border': 'horizontal' } }

    " Popup with 100% height
    let g:fzf_layout = { 'window': { 'width': 0.5, 'height': 1.0, 'border': 'vertical' } }

    " Similar to 'down' layout, but it uses a popup window and doesn't affect the window layout
    let g:fzf_layout = { 'window': { 'width': 1.0, 'height': 0.5, 'yoffset': 1.0, 'border': 'top' } }

    " Opens on the right;
    "   'highlight' option is still supported but it will only take the foreground color of the group
    let g:fzf_layout = { 'window': { 'width': 0.5, 'height': 1.0, 'xoffset': 1.0, 'border': 'left', 'highlight': 'Comment' } }
    ```
- To indicate if `--multi` mode is enabled, fzf will print the number of
  selected items even when no item is selected
  ```sh
  seq 100 | fzf
    # 100/100
  seq 100 | fzf --multi
    # 100/100 (0)
  seq 100 | fzf --multi 5
    # 100/100 (0/5)
  ```
- Since 0.24.0, release binaries will be uploaded to https://github.com/junegunn/fzf/releases

0.23.1
------
- Added `--preview-window` options for disabling flags
    - `nocycle`
    - `nohidden`
    - `nowrap`
    - `default`
- Built with Go 1.14.9 due to performance regression
    - https://github.com/golang/go/issues/40727

0.23.0
------
- Support preview scroll offset relative to window height
  ```sh
  git grep --line-number '' |
    fzf --delimiter : \
        --preview 'bat --style=numbers --color=always --highlight-line {2} {1}' \
        --preview-window +{2}-/2
  ```
- Added `--preview-window` option for sharp edges (`--preview-window sharp`)
- Added `--preview-window` option for cyclic scrolling (`--preview-window cycle`)
- Reduced vertical padding around the preview window when `--preview-window
  noborder` is used
- Added actions for preview window
    - `preview-half-page-up`
    - `preview-half-page-down`
- Vim
    - Popup width and height can be given in absolute integer values
    - Added `fzf#exec()` function for getting the path of fzf executable
        - It also downloads the latest binary if it's not available by running
          `./install --bin`
- Built with Go 1.15.2
    - We no longer provide 32-bit binaries

0.22.0
------
- Added more options for `--bind`
    - `backward-eof` event
      ```sh
      # Aborts when you delete backward when the query prompt is already empty
      fzf --bind backward-eof:abort
      ```
    - `refresh-preview` action
      ```sh
      # Rerun preview command when you hit '?'
      fzf --preview 'echo $RANDOM' --bind '?:refresh-preview'
      ```
    - `preview` action
      ```sh
      # Default preview command with an extra preview binding
      fzf --preview 'file {}' --bind '?:preview:cat {}'

      # A preview binding with no default preview command
      # (Preview window is initially empty)
      fzf --bind '?:preview:cat {}'

      # Preview window hidden by default, it appears when you first hit '?'
      fzf --bind '?:preview:cat {}' --preview-window hidden
      ```
- Added preview window option for setting the initial scroll offset
  ```sh
  # Initial scroll offset is set to the line number of each line of
  # git grep output *minus* 5 lines
  git grep --line-number '' |
    fzf --delimiter : --preview 'nl {1}' --preview-window +{2}-5
  ```
- Added support for ANSI colors in `--prompt` string
- Smart match of accented characters
    - An unaccented character in the query string will match both accented and
      unaccented characters, while an accented character will only match
      accented characters. This is similar to how "smart-case" match works.
- Vim plugin
    - `tmux` layout option for using fzf-tmux
      ```vim
      let g:fzf_layout = { 'tmux': '-p90%,60%' }
      ```

0.21.1
------
- Shell extension
    - CTRL-R will remove duplicate commands
- fzf-tmux
    - Supports tmux popup window (require tmux 3.2 or above)
        - ```sh
          # 50% width and height
          fzf-tmux -p

          # 80% width and height
          fzf-tmux -p 80%

          # 80% width and 40% height
          fzf-tmux -p 80%,40%
          fzf-tmux -w 80% -h 40%

          # Window position
          fzf-tmux -w 80% -h 40% -x 0 -y 0
          fzf-tmux -w 80% -h 40% -y 1000

          # Write ordinary fzf options after --
          fzf-tmux -p -- --reverse --info=inline --margin 2,4 --border
          ```
        - On macOS, you can build the latest tmux from the source with
          `brew install tmux --HEAD`
- Bug fixes
    - Fixed Windows file traversal not to include directories
    - Fixed ANSI colors with `--keep-right`
    - Fixed _fzf_complete for zsh
- Built with Go 1.14.1

0.21.0
------
- `--height` option is now available on Windows as well (@kelleyma49)
- Added `--pointer` and `--marker` options
- Added `--keep-right` option that keeps the right end of the line visible
  when it's too long
- Style changes
    - `--border` will now print border with rounded corners around the
      finder instead of printing horizontal lines above and below it.
      The previous style is available via `--border=horizontal`
    - Unicode spinner
- More keys and actions for `--bind`
- Added PowerShell script for downloading Windows binary
- Vim plugin: Built-in floating windows support
  ```vim
  let g:fzf_layout = { 'window': { 'width': 0.9, 'height': 0.6 } }
  ```
- bash: Various improvements in key bindings (CTRL-T, CTRL-R, ALT-C)
    - CTRL-R will start with the current command-line as the initial query
    - CTRL-R properly supports multi-line commands
- Fuzzy completion API changed
  ```sh
  # Previous: fzf arguments given as a single string argument
  # - This style is still supported, but it's deprecated
  _fzf_complete "--multi --reverse --prompt=\"doge> \"" "$@" < <(
    echo foo
  )

  # New API: multiple fzf arguments before "--"
  # - Easier to write multiple options
  _fzf_complete --multi --reverse --prompt="doge> " -- "$@" < <(
    echo foo
  )
  ```
- Bug fixes and improvements

0.20.0
------
- Customizable preview window color (`preview-fg` and `preview-bg` for `--color`)
  ```sh
  fzf --preview 'cat {}' \
      --color 'fg:#bbccdd,fg+:#ddeeff,bg:#334455,preview-bg:#223344,border:#778899' \
      --border --height 20 --layout reverse --info inline
  ```
- Removed the immediate flicking of the screen on `reload` action.
  ```sh
  : | fzf --bind 'change:reload:seq {q}' --phony
  ```
- Added `clear-query` and `clear-selection` actions for `--bind`
- It is now possible to split a composite bind action over multiple `--bind`
  expressions by prefixing the later ones with `+`.
  ```sh
  fzf --bind 'ctrl-a:up+up'

  # Can be now written as
  fzf --bind 'ctrl-a:up' --bind 'ctrl-a:+up'

  # This is useful when you need to write special execute/reload form (i.e. `execute:...`)
  # to avoid parse errors and add more actions to the same key
  fzf --multi --bind 'ctrl-l:select-all+execute:less {+f}' --bind 'ctrl-l:+deselect-all'
  ```
- Fixed parse error of `--bind` expression where concatenated execute/reload
  action contains `+` character.
  ```sh
  fzf --multi --bind 'ctrl-l:select-all+execute(less {+f})+deselect-all'
  ```
- Fixed bugs of reload action
    - Not triggered when there's no match even when the command doesn't have
      any placeholder expressions
    - Screen not properly cleared when `--header-lines` not filled on reload

0.19.0
------

- Added `--phony` option which completely disables search functionality.
  Useful when you want to use fzf only as a selector interface. See below.
- Added "reload" action for dynamically updating the input list without
  restarting fzf. See https://github.com/junegunn/fzf/issues/1750 to learn
  more about it.
  ```sh
  # Using fzf as the selector interface for ripgrep
  RG_PREFIX="rg --column --line-number --no-heading --color=always --smart-case "
  INITIAL_QUERY="foo"
  FZF_DEFAULT_COMMAND="$RG_PREFIX '$INITIAL_QUERY' || true" \
    fzf --bind "change:reload:$RG_PREFIX {q} || true" \
        --ansi --phony --query "$INITIAL_QUERY"
  ```
- `--multi` now takes an optional integer argument which indicates the maximum
  number of items that can be selected
  ```sh
  seq 100 | fzf --multi 3 --reverse --height 50%
  ```
- If a placeholder expression for `--preview` and `execute` action (and the
  new `reload` action) contains `f` flag, it is replaced to the
  path of a temporary file that holds the evaluated list. This is useful
  when you multi-select a large number of items and the length of the
  evaluated string may exceed [`ARG_MAX`][argmax].
  ```sh
  # Press CTRL-A to select 100K items and see the sum of all the numbers
  seq 100000 | fzf --multi --bind ctrl-a:select-all \
                   --preview "awk '{sum+=\$1} END {print sum}' {+f}"
  ```
- `deselect-all` no longer deselects unmatched items. It is now consistent
  with `select-all` and `toggle-all` in that it only affects matched items.
- Due to the limitation of bash, fuzzy completion is enabled by default for
  a fixed set of commands. A helper function for easily setting up fuzzy
  completion for any command is now provided.
  ```sh
  # usage: _fzf_setup_completion path|dir COMMANDS...
  _fzf_setup_completion path git kubectl
  ```
- Info line style can be changed by `--info=STYLE`
    - `--info=default`
    - `--info=inline` (same as old `--inline-info`)
    - `--info=hidden`
- Preview window border can be disabled by adding `noborder` to
  `--preview-window`.
- When you transform the input with `--with-nth`, the trailing white spaces
  are removed.
- `ctrl-\`, `ctrl-]`, `ctrl-^`, and `ctrl-/` can now be used with `--bind`
- See https://github.com/junegunn/fzf/milestone/15?closed=1 for more details

[argmax]: https://unix.stackexchange.com/questions/120642/what-defines-the-maximum-size-for-a-command-single-argument

0.18.0
------

- Added placeholder expression for zero-based item index: `{n}` and `{+n}`
    - `fzf --preview 'echo {n}: {}'`
- Added color option for the gutter: `--color gutter:-1`
- Added `--no-unicode` option for drawing borders in non-Unicode, ASCII
  characters
- `FZF_PREVIEW_LINES` and `FZF_PREVIEW_COLUMNS` are exported to preview process
    - fzf still overrides `LINES` and `COLUMNS` as before, but they may be
      reset by the default shell.
- Bug fixes and improvements
    - See https://github.com/junegunn/fzf/milestone/14?closed=1
- Built with Go 1.12.1

0.17.5
------

- Bug fixes and improvements
    - See https://github.com/junegunn/fzf/milestone/13?closed=1
- Search query longer than the screen width is allowed (up to 300 chars)
- Built with Go 1.11.1

0.17.4
------

- Added `--layout` option with a new layout called `reverse-list`.
    - `--layout=reverse` is a synonym for `--reverse`
    - `--layout=default` is a synonym for `--no-reverse`
- Preview window will be updated even when there is no match for the query
  if any of the placeholder expressions (e.g. `{q}`, `{+}`) evaluates to
  a non-empty string.
- More keys for binding: `shift-{up,down}`, `alt-{up,down,left,right}`
- fzf can now start even when `/dev/tty` is not available by making an
  educated guess.
- Updated the default command for Windows.
- Fixes and improvements on bash/zsh completion
- install and uninstall scripts now supports generating files under
  `XDG_CONFIG_HOME` on `--xdg` flag.

See https://github.com/junegunn/fzf/milestone/12?closed=1 for the full list of
changes.

0.17.3
------
- `$LINES` and `$COLUMNS` are exported to preview command so that the command
  knows the exact size of the preview window.
- Better error messages when the default command or `$FZF_DEFAULT_COMMAND`
  fails.
- Reverted #1061 to avoid having duplicate entries in the list when find
  command detected a file system loop (#1120). The default command now
  requires that find supports `-fstype` option.
- fzf now distinguishes mouse left click and right click (#1130)
    - Right click is now bound to `toggle` action by default
    - `--bind` understands `left-click` and `right-click`
- Added `replace-query` action (#1137)
    - Replaces query string with the current selection
- Added `accept-non-empty` action (#1162)
    - Same as accept, except that it prevents fzf from exiting without any
      selection

0.17.1
------

- Fixed custom background color of preview window (#1046)
- Fixed background color issues of Windows binary
- Fixed Windows binary to execute command using cmd.exe with no parsing and
  escaping (#1072)
- Added support for `window` layout on Vim 8 using Vim 8 terminal (#1055)

0.17.0-2
--------

A maintenance release for auxiliary scripts. fzf binaries are not updated.

- Experimental support for the builtin terminal of Vim 8
    - fzf can now run inside GVim
- Updated Vim plugin to better handle `&shell` issue on fish
- Fixed a bug of fzf-tmux where invalid output is generated
- Fixed fzf-tmux to work even when `tput` does not work

0.17.0
------
- Performance optimization
- One can match literal spaces in extended-search mode with a space prepended
  by a backslash.
- `--expect` is now additive and can be specified multiple times.

0.16.11
-------
- Performance optimization
- Fixed missing preview update

0.16.10
-------
- Fixed invalid handling of ANSI colors in preview window
- Further improved `--ansi` performance

0.16.9
------
- Memory and performance optimization
    - Around 20% performance improvement for general use cases
    - Up to 5x faster processing of `--ansi`
    - Up to 50% reduction of memory usage
- Bug fixes and usability improvements
    - Fixed handling of bracketed paste mode
    - [ERROR] on info line when the default command failed
    - More efficient rendering of preview window
    - `--no-clear` updated for repetitive relaunching scenarios

0.16.8
------
- New `change` event and `top` action for `--bind`
    - `fzf --bind change:top`
        - Move cursor to the top result whenever the query string is changed
    - `fzf --bind 'ctrl-w:unix-word-rubout+top,ctrl-u:unix-line-discard+top'`
        - `top` combined with `unix-word-rubout` and `unix-line-discard`
- Fixed inconsistent tiebreak scores when `--nth` is used
- Proper display of tab characters in `--prompt`
- Fixed not to `--cycle` on page-up/page-down to prevent overshoot
- Git revision in `--version` output
- Basic support for Cygwin environment
- Many fixes in Vim plugin on Windows/Cygwin (thanks to @janlazo)

0.16.7
------
- Added support for `ctrl-alt-[a-z]` key chords
- CTRL-Z (SIGSTOP) now works with fzf
- fzf will export `$FZF_PREVIEW_WINDOW` so that the scripts can use it
- Bug fixes and improvements in Vim plugin and shell extensions

0.16.6
------
- Minor bug fixes and improvements
- Added `--no-clear` option for scripting purposes

0.16.5
------
- Minor bug fixes
- Added `toggle-preview-wrap` action
- Built with Go 1.8

0.16.4
------
- Added `--border` option to draw border above and below the finder
- Bug fixes and improvements

0.16.3
------
- Fixed a bug where fzf incorrectly display the lines when straddling tab
  characters are trimmed
- Placeholder expression used in `--preview` and `execute` action can
  optionally take `+` flag to be used with multiple selections
    - e.g. `git log --oneline | fzf --multi --preview 'git show {+1}'`
- Added `execute-silent` action for executing a command silently without
  switching to the alternate screen. This is useful when the process is
  short-lived and you're not interested in its output.
    - e.g. `fzf --bind 'ctrl-y:execute!(echo -n {} | pbcopy)'`
- `ctrl-space` is allowed in `--bind`

0.16.2
------
- Dropped ncurses dependency
- Binaries for freebsd, openbsd, arm5, arm6, arm7, and arm8
- Official 24-bit color support
- Added support for composite actions in `--bind`. Multiple actions can be
  chained using `+` separator.
    - e.g. `fzf --bind 'ctrl-y:execute(echo -n {} | pbcopy)+abort'`
- `--preview-window` with size 0 is allowed. This is used to make fzf execute
  preview command in the background without displaying the result.
- Minor bug fixes and improvements

0.16.1
------
- Fixed `--height` option to properly fill the window with the background
  color
- Added `half-page-up` and `half-page-down` actions
- Added `-L` flag to the default find command

0.16.0
------
- *Added `--height HEIGHT[%]` option*
    - fzf can now display finder without occupying the full screen
- Preview window will truncate long lines by default. Line wrap can be enabled
  by `:wrap` flag in `--preview-window`.
- Latin script letters will be normalized before matching so that it's easier
  to match against accented letters. e.g. `sodanco` can match `Só Danço Samba`.
    - Normalization can be disabled via `--literal`
- Added `--filepath-word` to make word-wise movements/actions (`alt-b`,
  `alt-f`, `alt-bs`, `alt-d`) respect path separators

0.15.9
------
- Fixed rendering glitches introduced in 0.15.8
- The default escape delay is reduced to 50ms and is configurable via
  `$ESCDELAY`
- Scroll indicator at the top-right corner of the preview window is always
  displayed when there's overflow
- Can now be built with ncurses 6 or tcell to support extra features
    - *ncurses 6*
        - Supports more than 256 color pairs
        - Supports italics
    - *tcell*
        - 24-bit color support
    - See https://github.com/junegunn/fzf/blob/master/BUILD.md

0.15.8
------
- Updated ANSI processor to handle more VT-100 escape sequences
- Added `--no-bold` (and `--bold`) option
- Improved escape sequence processing for WSL
- Added support for `alt-[0-9]`, `f11`, and `f12` for `--bind` and `--expect`

0.15.7
------
- Fixed panic when color is disabled and header lines contain ANSI colors

0.15.6
------
- Windows binaries! (@kelleyma49)
- Fixed the bug where header lines are cleared when preview window is toggled
- Fixed not to display ^N and ^O on screen
- Fixed cursor keys (or any key sequence that starts with ESC) on WSL by
  making fzf wait for additional keystrokes after ESC for up to 100ms

0.15.5
------
- Setting foreground color will no longer set background color to black
    - e.g. `fzf --color fg:153`
- `--tiebreak=end` will consider relative position instead of absolute distance
- Updated `fzf#wrap` function to respect `g:fzf_colors`

0.15.4
------
- Added support for range expression in preview and execute action
    - e.g. `ls -l | fzf --preview="echo user={3} when={-4..-2}; cat {-1}" --header-lines=1`
    - `{q}` will be replaced to the single-quoted string of the current query
- Fixed to properly handle unicode whitespace characters
- Display scroll indicator in preview window
- Inverse search term will use exact matcher by default
    - This is a breaking change, but I believe it makes much more sense. It is
      almost impossible to predict which entries will be filtered out due to
      a fuzzy inverse term. You can still perform inverse-fuzzy-match by
      prepending `!'` to the term.

0.15.3
------
- Added support for more ANSI attributes: dim, underline, blink, and reverse
- Fixed race condition in `toggle-preview`

0.15.2
------
- Preview window is now scrollable
    - With mouse scroll or with bindable actions
        - `preview-up`
        - `preview-down`
        - `preview-page-up`
        - `preview-page-down`
- Updated ANSI processor to support high intensity colors and ignore
  some VT100-related escape sequences

0.15.1
------
- Fixed panic when the pattern occurs after 2^15-th column
- Fixed rendering delay when displaying extremely long lines

0.15.0
------
- Improved fuzzy search algorithm
    - Added `--algo=[v1|v2]` option so one can still choose the old algorithm
      which values the search performance over the quality of the result
- Advanced scoring criteria
- `--read0` to read input delimited by ASCII NUL character
- `--print0` to print output delimited by ASCII NUL character

0.13.5
------
- Memory and performance optimization
    - Up to 2x performance with half the amount of memory

0.13.4
------
- Performance optimization
    - Memory footprint for ascii string is reduced by 60%
    - 15 to 20% improvement of query performance
    - Up to 45% better performance of `--nth` with non-regex delimiters
- Fixed invalid handling of `hidden` property of `--preview-window`

0.13.3
------
- Fixed duplicate rendering of the last line in preview window

0.13.2
------
- Fixed race condition where preview window is not properly cleared

0.13.1
------
- Fixed UI issue with large `--preview` output with many ANSI codes

0.13.0
------
- Added preview feature
    - `--preview CMD`
    - `--preview-window POS[:SIZE][:hidden]`
- `{}` in execute action is now replaced to the single-quoted (instead of
  double-quoted) string of the current line
- Fixed to ignore control characters for bracketed paste mode

0.12.2
------

- 256-color capability detection does not require `256` in `$TERM`
- Added `print-query` action
- More named keys for binding; <kbd>F1</kbd> ~ <kbd>F10</kbd>,
  <kbd>ALT-/</kbd>, <kbd>ALT-space</kbd>, and <kbd>ALT-enter</kbd>
- Added `jump` and `jump-accept` actions that implement [EasyMotion][em]-like
  movement
  ![][jump]

[em]: https://github.com/easymotion/vim-easymotion
[jump]: https://cloud.githubusercontent.com/assets/700826/15367574/b3999dc4-1d64-11e6-85da-28ceeb1a9bc2.png

0.12.1
------

- Ranking algorithm introduced in 0.12.0 is now universally applied
- Fixed invalid cache reference in exact mode
- Fixes and improvements in Vim plugin and shell extensions

0.12.0
------

- Enhanced ranking algorithm
- Minor bug fixes

0.11.4
------

- Added `--hscroll-off=COL` option (default: 10) (#513)
- Some fixes in Vim plugin and shell extensions

0.11.3
------

- Graceful exit on SIGTERM (#482)
- `$SHELL` instead of `sh` for `execute` action and `$FZF_DEFAULT_COMMAND` (#481)
- Changes in fuzzy completion API
    - [`_fzf_compgen_{path,dir}`](https://github.com/junegunn/fzf/commit/9617647)
    - [`_fzf_complete_COMMAND_post`](https://github.com/junegunn/fzf/commit/8206746)
      for post-processing

0.11.2
------

- `--tiebreak` now accepts comma-separated list of sort criteria
    - Each criterion should appear only once in the list
    - `index` is only allowed at the end of the list
    - `index` is implicitly appended to the list when not specified
    - Default is `length` (or equivalently `length,index`)
- `begin` criterion will ignore leading whitespaces when calculating the index
- Added `toggle-in` and `toggle-out` actions
    - Switch direction depending on `--reverse`-ness
    - `export FZF_DEFAULT_OPTS="--bind tab:toggle-out,shift-tab:toggle-in"`
- Reduced the initial delay when `--tac` is not given
    - fzf defers the initial rendering of the screen up to 100ms if the input
      stream is ongoing to prevent unnecessary redraw during the initial
      phase. However, 100ms delay is quite noticeable and might give the
      impression that fzf is not snappy enough. This commit reduces the
      maximum delay down to 20ms when `--tac` is not specified, in which case
      the input list quickly fills the entire screen.

0.11.1
------

- Added `--tabstop=SPACES` option

0.11.0
------

- Added OR operator for extended-search mode
- Added `--execute-multi` action
- Fixed incorrect cursor position when unicode wide characters are used in
  `--prompt`
- Fixes and improvements in shell extensions

0.10.9
------

- Extended-search mode is now enabled by default
    - `--extended-exact` is deprecated and instead we have `--exact` for
      orthogonally controlling "exactness" of search
- Fixed not to display non-printable characters
- Added `double-click` for `--bind` option
- More robust handling of SIGWINCH

0.10.8
------

- Fixed panic when trying to set colors after colors are disabled (#370)

0.10.7
------

- Fixed unserialized interrupt handling during execute action which often
  caused invalid memory access and crash
- Changed `--tiebreak=length` (default) to use trimmed length when `--nth` is
  used

0.10.6
------

- Replaced `--header-file` with `--header` option
- `--header` and `--header-lines` can be used together
- Changed exit status
    - 0: Okay
    - 1: No match
    - 2: Error
    - 130: Interrupted
- 64-bit linux binary is statically-linked with ncurses to avoid
  compatibility issues.

0.10.5
------

- `'`-prefix to unquote the term in `--extended-exact` mode
- Backward scan when `--tiebreak=end` is set

0.10.4
------

- Fixed to remove ANSI code from output when `--with-nth` is set

0.10.3
------

- Fixed slow performance of `--with-nth` when used with `--delimiter`
    - Regular expression engine of Golang as of now is very slow, so the fixed
      version will treat the given delimiter pattern as a plain string instead
      of a regular expression unless it contains special characters and is
      a valid regular expression.
    - Simpler regular expression for delimiter for better performance

0.10.2
------

### Fixes and improvements

- Improvement in perceived response time of queries
    - Eager, efficient rune array conversion
- Graceful exit when failed to initialize ncurses (invalid $TERM)
- Improved ranking algorithm when `--nth` option is set
- Changed the default command not to fail when there are files whose names
  start with dash

0.10.1
------

### New features

- Added `--margin` option
- Added options for sticky header
    - `--header-file`
    - `--header-lines`
- Added `cancel` action which clears the input or closes the finder when the
  input is already empty
    - e.g. `export FZF_DEFAULT_OPTS="--bind esc:cancel"`
- Added `delete-char/eof` action to differentiate `CTRL-D` and `DEL`

### Minor improvements/fixes

- Fixed to allow binding colon and comma keys
- Fixed ANSI processor to handle color regions spanning multiple lines

0.10.0
------

### New features

- More actions for `--bind`
    - `select-all`
    - `deselect-all`
    - `toggle-all`
    - `ignore`
- `execute(...)` action for running arbitrary command without leaving fzf
    - `fzf --bind "ctrl-m:execute(less {})"`
    - `fzf --bind "ctrl-t:execute(tmux new-window -d 'vim {}')"`
    - If the command contains parentheses, use any of the follows alternative
      notations to avoid parse errors
        - `execute[...]`
        - `execute~...~`
        - `execute!...!`
        - `execute@...@`
        - `execute#...#`
        - `execute$...$`
        - `execute%...%`
        - `execute^...^`
        - `execute&...&`
        - `execute*...*`
        - `execute;...;`
        - `execute/.../`
        - `execute|...|`
        - `execute:...`
            - This is the special form that frees you from parse errors as it
              does not expect the closing character
            - The catch is that it should be the last one in the
              comma-separated list
- Added support for optional search history
    - `--history HISTORY_FILE`
        - When used, `CTRL-N` and `CTRL-P` are automatically remapped to
          `next-history` and `previous-history`
    - `--history-size MAX_ENTRIES` (default: 1000)
- Cyclic scrolling can be enabled with `--cycle`
- Fixed the bug where the spinner was not spinning on idle input stream
    - e.g. `sleep 100 | fzf`

### Minor improvements/fixes

- Added synonyms for key names that can be specified for `--bind`,
  `--toggle-sort`, and `--expect`
- Fixed the color of multi-select marker on the current line
- Fixed to allow `^pattern$` in extended-search mode


0.9.13
------

### New features

- Color customization with the extended `--color` option

### Bug fixes

- Fixed premature termination of Reader in the presence of a long line which
  is longer than 64KB

0.9.12
------

### New features

- Added `--bind` option for custom key bindings

### Bug fixes

- Fixed to update "inline-info" immediately after terminal resize
- Fixed ANSI code offset calculation

0.9.11
------

### New features

- Added `--inline-info` option for saving screen estate (#202)
     - Useful inside Neovim
     - e.g. `let $FZF_DEFAULT_OPTS = $FZF_DEFAULT_OPTS.' --inline-info'`

### Bug fixes

- Invalid mutation of input on case conversion (#209)
- Smart-case for each term in extended-search mode (#208)
- Fixed double-click result when scroll offset is positive

0.9.10
------

### Improvements

- Performance optimization
- Less aggressive memoization to limit memory usage

### New features

- Added color scheme for light background: `--color=light`

0.9.9
-----

### New features

- Added `--tiebreak` option (#191)
- Added `--no-hscroll` option (#193)
- Visual indication of `--toggle-sort` (#194)

0.9.8
-----

### Bug fixes

- Fixed Unicode case handling (#186)
- Fixed to terminate on RuneError (#185)

0.9.7
-----

### New features

- Added `--toggle-sort` option (#173)
    - `--toggle-sort=ctrl-r` is applied to `CTRL-R` shell extension

### Bug fixes

- Fixed to print empty line if `--expect` is set and fzf is completed by
  `--select-1` or `--exit-0` (#172)
- Fixed to allow comma character as an argument to `--expect` option

0.9.6
-----

### New features

#### Added `--expect` option (#163)

If you provide a comma-separated list of keys with `--expect` option, fzf will
allow you to select the match and complete the finder when any of the keys is
pressed. Additionally, fzf will print the name of the key pressed as the first
line of the output so that your script can decide what to do next based on the
information.

```sh
fzf --expect=ctrl-v,ctrl-t,alt-s,f1,f2,~,@
```

The updated vim plugin uses this option to implement
[ctrlp](https://github.com/kien/ctrlp.vim)-compatible key bindings.

### Bug fixes

- Fixed to ignore ANSI escape code `\e[K` (#162)

0.9.5
-----

### New features

#### Added `--ansi` option (#150)

If you give `--ansi` option to fzf, fzf will interpret ANSI color codes from
the input, display the item with the ANSI colors (true colors are not
supported), and strips the codes from the output. This option is off by
default as it entails some overhead.

### Improvements

#### Reduced initial memory footprint (#151)

By removing unnecessary copy of pointers, fzf will use significantly smaller
amount of memory when it's started. The difference is hugely noticeable when
the input is extremely large. (e.g. `locate / | fzf`)

### Bug fixes

- Fixed panic on `--no-sort --filter ''` (#149)

0.9.4
-----

### New features

#### Added `--tac` option to reverse the order of the input.

One might argue that this option is unnecessary since we can already put `tac`
or `tail -r` in the command pipeline to achieve the same result. However, the
advantage of `--tac` is that it does not block until the input is complete.

### *Backward incompatible changes*

#### Changed behavior on `--no-sort`

`--no-sort` option will no longer reverse the display order within finder. You
may want to use the new `--tac` option with `--no-sort`.

```
history | fzf +s --tac
```

### Improvements

#### `--filter` will not block when sort is disabled

When fzf works in filtering mode (`--filter`) and sort is disabled
(`--no-sort`), there's no need to block until input is complete. The new
version of fzf will print the matches on-the-fly when the following condition
is met:

    --filter TERM --no-sort [--no-tac --no-sync]

or simply:

    -f TERM +s

This change removes unnecessary delay in the use cases like the following:

    fzf -f xxx +s | head -5

However, in this case, fzf processes the lines sequentially, so it cannot
utilize multiple cores, and fzf will run slightly slower than the previous
mode of execution where filtering is done in parallel after the entire input
is loaded. If the user is concerned about this performance problem, one can
add `--sync` option to re-enable buffering.

0.9.3
-----

### New features
- Added `--sync` option for multi-staged filtering

### Improvements
- `--select-1` and `--exit-0` will start finder immediately when the condition
  cannot be met

