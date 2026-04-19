#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ key-bindings.nu
#
# - $FZF_TMUX               (default: 0)
# - $FZF_TMUX_OPTS
# - $FZF_TMUX_HEIGHT        (default: 40%)
# - $FZF_CTRL_T_COMMAND     (set to "" to disable)
# - $FZF_CTRL_T_OPTS
# - $FZF_CTRL_R_COMMAND     (set to "" to disable)
# - $FZF_CTRL_R_OPTS
# - $FZF_ALT_C_COMMAND      (set to "" to disable)
# - $FZF_ALT_C_OPTS

# Code provided by @igor-ramazanov
# Source: https://github.com/junegunn/fzf/issues/4122#issuecomment-2607368316

# Merge default options in the same order as bash/zsh:
#   1. --height, --min-height, --bind=ctrl-z:ignore, $prepend
#   2. $FZF_DEFAULT_OPTS_FILE contents
#   3. $FZF_DEFAULT_OPTS, $append
def __fzf_defaults [prepend: string, append: string]: nothing -> string {
  let base = $"--height ($env.FZF_TMUX_HEIGHT? | default '40%') --min-height 20+ --bind=ctrl-z:ignore ($prepend)"
  let opts_file = if ($env.FZF_DEFAULT_OPTS_FILE? | default '' | is-not-empty) {
    try { open --raw ($env.FZF_DEFAULT_OPTS_FILE) | str trim } catch { '' }
  } else {
    ''
  }
  let default_opts = $env.FZF_DEFAULT_OPTS? | default ''
  $"($base) ($opts_file) ($default_opts) ($append)" | str trim
}

# Return the fzf command to use: fzf-tmux when inside tmux and
# FZF_TMUX is enabled or FZF_TMUX_OPTS is set, plain fzf otherwise.
def __fzfcmd []: nothing -> list<string> {
  let in_tmux = ($env.TMUX_PANE? | default '' | into string | is-not-empty)
  if $in_tmux {
    let fzf_tmux = ($env.FZF_TMUX? | default 0 | into string)
    let fzf_tmux_opts = ($env.FZF_TMUX_OPTS? | default '' | into string)
    if ($fzf_tmux != '0') or ($fzf_tmux_opts | is-not-empty) {
      let opts = if ($fzf_tmux_opts | is-not-empty) { $fzf_tmux_opts } else { $"-d($env.FZF_TMUX_HEIGHT? | default '40%')" }
      return ['fzf-tmux' ...(($opts | split row ' ' | where { $in != '' })) '--']
    }
  }
  ['fzf']
}


export-env {
  $env.FZF_CTRL_T_OPTS     = $env.FZF_CTRL_T_OPTS?     | default ""
  $env.FZF_CTRL_R_OPTS     = $env.FZF_CTRL_R_OPTS?     | default ""
  $env.FZF_ALT_C_OPTS      = $env.FZF_ALT_C_OPTS?      | default ""
}

# Directories
const alt_c = {
    name: fzf_dirs
    modifier: alt
    keycode: char_c
    mode: [emacs, vi_normal, vi_insert]
    event: [
      {
        send: executehostcommand
        cmd: "
          let fzf_opts = (__fzf_defaults '--reverse --walker=dir,follow,hidden --scheme=path' $'($env.FZF_ALT_C_OPTS) +m');
          let fzfcmd = (__fzfcmd);
          let fzf_args = ($fzfcmd | skip 1);
          let alt_c_cmd = ($env.FZF_ALT_C_COMMAND? | default null);
          let result = if ($alt_c_cmd == null) or ($alt_c_cmd | is-empty) {
            with-env { FZF_DEFAULT_OPTS: $fzf_opts, FZF_DEFAULT_OPTS_FILE: '' } { ^($fzfcmd | first) ...$fzf_args }
          } else {
            let fzf_cmd_str = ($fzfcmd | str join ' ');
            with-env { FZF_DEFAULT_OPTS: $fzf_opts, FZF_DEFAULT_OPTS_FILE: '' } { nu -c $'($alt_c_cmd) | ($fzf_cmd_str)' }
          };
          cd $result;
        "
      }
    ]
}

# History
const ctrl_r = {
  name: history_menu
  modifier: control
  keycode: char_r
  mode: [emacs, vi_insert, vi_normal]
  event: [
    {
      send: executehostcommand
      cmd: "commandline edit --replace (
        let fzf_opts = (__fzf_defaults '' $'--scheme=history --bind=ctrl-r:toggle-sort --wrap-sign \"\t↳ \" --highlight-line ($env.FZF_CTRL_R_OPTS) +m --read0');
        let fzfcmd = (__fzfcmd);
        let fzf_args = ($fzfcmd | skip 1);
        history
          | get command
          | reverse
          | uniq
          | str join (char -i 0)
          | with-env { FZF_DEFAULT_OPTS: $fzf_opts, FZF_DEFAULT_OPTS_FILE: '' } { ^($fzfcmd | first) ...$fzf_args --query (commandline) }
          | decode utf-8
          | str trim
      )"
    }
  ]
}

# Files
const ctrl_t =  {
    name: fzf_files
    modifier: control
    keycode: char_t
    mode: [emacs, vi_normal, vi_insert]
    event: [
      {
        send: executehostcommand
        cmd: "
          let fzf_opts = (__fzf_defaults '--reverse --walker=file,dir,follow,hidden --scheme=path' $'($env.FZF_CTRL_T_OPTS) -m');
          let fzfcmd = (__fzfcmd);
          let fzf_args = ($fzfcmd | skip 1);
          let ctrl_t_cmd = ($env.FZF_CTRL_T_COMMAND? | default null);
          let result = if ($ctrl_t_cmd == null) or ($ctrl_t_cmd | is-empty) {
            with-env { FZF_DEFAULT_OPTS: $fzf_opts, FZF_DEFAULT_OPTS_FILE: '' } { ^($fzfcmd | first) ...$fzf_args }
          } else {
            let fzf_cmd_str = ($fzfcmd | str join ' ');
            with-env { FZF_DEFAULT_OPTS: $fzf_opts, FZF_DEFAULT_OPTS_FILE: '' } { nu -l -i -c $'($ctrl_t_cmd) | ($fzf_cmd_str)' }
          };
          commandline edit --append $result;
          commandline set-cursor --end
        "
      }
    ]
}

# Helper to check if a binding is enabled. A binding is disabled when
# the corresponding *_COMMAND variable is explicitly set to "".
# When not defined (null), the binding is enabled (using fzf's built-in walker).
def __fzf_binding_enabled [var_name: string]: nothing -> bool {
  let val = ($env | get -o $var_name)
  # null = not defined = enabled; "" = explicitly disabled
  $val == null or ($val | into string | is-not-empty)
}

# Update the $env.config
export-env {
  let already_loaded = ($env.config.keybindings | any { |kb| $kb.name == 'fzf_files' })
  if not $already_loaded {
    mut bindings = []
    if (__fzf_binding_enabled 'FZF_ALT_C_COMMAND') { $bindings = ($bindings | append $alt_c) }
    if (__fzf_binding_enabled 'FZF_CTRL_R_COMMAND') { $bindings = ($bindings | append $ctrl_r) }
    if (__fzf_binding_enabled 'FZF_CTRL_T_COMMAND') { $bindings = ($bindings | append $ctrl_t) }
    $env.config.keybindings = ($env.config.keybindings | append $bindings)
  }
}
