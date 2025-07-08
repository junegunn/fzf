#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ key-bindings.nu
#
# - $FZF_TMUX_OPTS --
# - $FZF_CTRL_T_COMMAND
# - $FZF_CTRL_T_OPTS
# - $FZF_CTRL_R_OPTS ---
# - $FZF_ALT_C_COMMAND
# - $FZF_ALT_C_OPTS

# Dependencies: `fd`, `bat, `rg`, `nufmt`, `tree`.

# Code provided by @igor-ramazanov
# Source: https://github.com/junegunn/fzf/issues/4122#issuecomment-2607368316


export-env {
  $env.FZF_TMUX_OPTS = "--height 40%"
  $env.FZF_CTRL_T_COMMAND = "fd --type file --hidden"
  $env.FZF_CTRL_T_OPTS = "--preview 'bat --color=always --style=full --line-range=:500 {}' "
  $env.FZF_CTRL_R_OPTS = "" # Options for history search
  $env.FZF_ALT_C_COMMAND = "fd --type directory --hidden"
  $env.FZF_ALT_C_OPTS = "--preview 'tree -C {} | head -n 200'"
  $env.FZF_DEFAULT_COMMAND = "fd --type file --hidden"
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
          let fzf_alt_c_command = \$\"($env.FZF_ALT_C_COMMAND) | fzf ($env.FZF_ALT_C_OPTS)\";
          let result = nu -c $fzf_alt_c_command;
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
      cmd: "
        let result = history
          | get command
          | str replace --all (char newline) ' '
          | to text
          | fzf --preview 'printf \'{}\' | nufmt --stdin 2>&1 | rg -v ERROR';
        commandline edit --append $result;
        commandline set-cursor --end
      "
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
          let fzf_ctrl_t_command = \$\"($env.FZF_CTRL_T_COMMAND) | fzf ($env.FZF_CTRL_T_OPTS)\";
          let result = nu -l -i -c $fzf_ctrl_t_command;
          commandline edit --append $result;
          commandline set-cursor --end
        "
      }
    ]
}

# Update the $env.config
export-env {
  if not ($env.__keybindings_loaded? | default false) {
    $env.__keybindings_loaded = true
    $env.config.keybindings = $env.config.keybindings | append [
      $alt_c
      $ctrl_r
      $ctrl_t
    ]
  }
}