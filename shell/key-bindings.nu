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

# Code provided by @igor-ramazanov
# Source: https://github.com/junegunn/fzf/issues/4122#issuecomment-2607368316


export-env {
  $env.FZF_TMUX_OPTS       = $env.FZF_TMUX_OPTS?       | default "--height 40%"
  $env.FZF_CTRL_T_COMMAND  = $env.FZF_CTRL_T_COMMAND?  | default ""
  $env.FZF_CTRL_T_OPTS     = $env.FZF_CTRL_T_OPTS?     | default ""
  $env.FZF_CTRL_R_OPTS     = $env.FZF_CTRL_R_OPTS?     | default ""
  $env.FZF_ALT_C_COMMAND   = $env.FZF_ALT_C_COMMAND?   | default ""
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
          let fzf_opts = $'--reverse --walker=dir,follow,hidden --scheme=path ($env.FZF_ALT_C_OPTS) +m';
          let result = if ($env.FZF_ALT_C_COMMAND | is-empty) {
            fzf ...(($fzf_opts | split row ' ') | where { $in != '' })
          } else {
            let fzf_command = $'($env.FZF_ALT_C_COMMAND) | fzf ($fzf_opts)';
            nu -c $fzf_command
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
      cmd: "commandline edit --insert (
        let fzf_command = \$\"fzf --scheme=history --bind=ctrl-r:toggle-sort --wrap-sign '\t↳ ' --highlight-line --read0 --query '\(commandline\)' ($env.FZF_CTRL_R_OPTS) +m\";
        history
          | get command
          | reverse
          | uniq
          | str join (char -i 0)
          | nu -l -i -c $fzf_command
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
          let fzf_opts = $'--reverse --walker=file,dir,follow,hidden --scheme=path ($env.FZF_CTRL_T_OPTS) -m';
          let result = if ($env.FZF_CTRL_T_COMMAND | is-empty) {
            fzf ...(($fzf_opts | split row ' ') | where { $in != '' })
          } else {
            let fzf_command = $'($env.FZF_CTRL_T_COMMAND) | fzf ($fzf_opts)';
            nu -l -i -c $fzf_command
          };
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
