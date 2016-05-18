CHANGELOG
=========

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

