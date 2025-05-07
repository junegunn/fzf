#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ completion-menu.nu


# A different implementation aproach than completion.nu
# This tries to load FZF as a menu, and tries to implement
# a fallback using the until event type.
# https://www.nushell.sh/book/line_editor.html#until-type

# Unfortunately this currently doesn't work.
# See: https://github.com/nushell/reedline/issues/876

# For now the recomendation is to use completion-external.nu
# https://github.com/imsys/fzf/blob/master/shell/completion-external.nu


# --- Default Environment Variables ---
# These can be overridden in your config.nu or environment.
# Example: $env.FZF_COMPLETION_TRIGGER = "!<TAB>"

# - $env.FZF_TMUX                 (default: 0)
# - $env.FZF_TMUX_OPTS            (default: empty)
# - $env.FZF_TMUX_HEIGHT          (default: 40%)
# - $env.FZF_COMPLETION_TRIGGER   (default: '**')
# - $env.FZF_COMPLETION_OPTS      (default: empty)
# - $env.FZF_COMPLETION_PATH_OPTS (default: empty)
# - $env.FZF_COMPLETION_DIR_OPTS  (default: empty)


# Set default height for fzf-tmux pane. e.g. '40%'
$env.FZF_TMUX_HEIGHT = $env.FZF_TMUX_HEIGHT? | default '40%'
# Options for fzf-tmux wrapper. e.g. '--paneid popup'
$env.FZF_TMUX_OPTS = $env.FZF_TMUX_OPTS? | default ''

$env.FZF_COMPLETION_TRIGGER = $env.FZF_COMPLETION_TRIGGER? | default '**'

# Options for fzf completion in general. e.g. '--border'
$env.FZF_COMPLETION_OPTS = $env.FZF_COMPLETION_OPTS? | default ''

# Options specific to path completion. e.g. '--extended'
$env.FZF_COMPLETION_PATH_OPTS = $env.FZF_COMPLETION_PATH_OPTS? | default ''
# Options specific to directory completion. e.g. '--extended'
$env.FZF_COMPLETION_DIR_OPTS = $env.FZF_COMPLETION_DIR_OPTS? | default ''

$env.FZF_COMPLETION_DIR_COMMANDS = $env.FZF_COMPLETION_DIR_COMMANDS? | default ['cd', 'pushd', 'rmdir']
$env.FZF_COMPLETION_VAR_COMMANDS = $env.FZF_COMPLETION_VAR_COMMANDS? | default ['export', 'unset', 'printenv']

# --- Helper Functions ---

# Helper to build default fzf options list
def __fzf_defaults_nu [prepend: list<string>, append: string] {
  let default_opts = $env.FZF_DEFAULT_OPTS? | default ''
  let default_opts_file = $env.FZF_DEFAULT_OPTS_FILE? | default ''

  let file_opts = try {
     open $default_opts_file | lines | str trim | where not ($in | is-empty)
  } catch {
     [] # Return empty list on error (e.g., file not found)
  }

  # Build options list
  return $prepend # Start with the prepend argument
  | append $file_opts # Append options from file
  | append ($default_opts | split words | where not ($in | is-empty)) # Append options from $FZF_DEFAULT_OPTS
  | append ($append | split words | where not ($in | is-empty)) # Append options from function argument
  | where {|it| try { ($it | is-string) and not ($it | is-empty) } catch { false } } # Filter to keep only non-empty strings, safely handling potential errors
}

# Wrapper for running fzf or fzf-tmux
def __fzf_comprun_nu [
  context_name: string, # e.g., "fzf-completion", "fzf-helper" - mainly for potential debugging
  query: string,        # The initial query string for fzf
  fzf_opts_arg: list<string> # Remaining options for fzf/fzf-tmux
  ] {

  # in which case $stdin_content will be null.
  let stdin_content = try {
    # Collect stdin into a single string. Adjust if structured data is expected.
    $in | into string
  } catch {
    null # Set to null if there's no stdin or an error occurs reading it
  }

  let fzf_prefinal_opt = ['--query', $query, '--reverse'] 
    | append (__fzf_defaults_nu $fzf_opts_arg ($env.FZF_COMPLETION_OPTS | default ''))

  # Get the configured height, defaulting to '40%'
  let height_opt = $env.FZF_TMUX_HEIGHT? | default '40%'

  # Determine if fzf should generate its own candidates via walker
  let has_walker = ($fzf_prefinal_opt | find '--walker' | is-not-empty)

  # Check for custom comprun function (Nu equivalent)
  if ((help commands | where name == '_fzf_comprun_nu') | is-not-empty) {
    # Note: Nushell doesn't have a direct equivalent to Zsh/Bash `type -t _fzf_comprun`.
    # This check assumes a user might define a custom command named `_fzf_comprun_nu`.
    _fzf_comprun_nu $context_name $query ...$fzf_prefinal_opt # Pass args correctly to custom function
  } else if ($env.TMUX_PANE? | is-not-empty) and (($env.FZF_TMUX? | default 0) != 0 or ($env.FZF_TMUX_OPTS? | is-not-empty)) {
    # Running inside tmux, use fzf-tmux
    # Skip the first arg which is cmd_word (passed for context but not needed by fzf/fzf-tmux itself)
    let final_fzf_inner_opts =  $fzf_prefinal_opt

    let final_fzf_opts = if ($env.FZF_TMUX_OPTS? | is-not-empty) {
      $env.FZF_TMUX_OPTS | split row ' ' | append ['--'] | append $fzf_prefinal_opt
    } else {
      # Use the default -d option with the configured height for fzf-tmux
      ['-d', $height_opt, '--'] | append $fzf_prefinal_opt
    }
    
    if $has_walker or ($stdin_content == null) {
      # Run directly if walker or no stdin provided
      fzf-tmux ...$final_fzf_opts
    } else {
      # Pipe captured stdin to fzf-tmux
      $stdin_content | fzf-tmux ...$final_fzf_opts
    }
    
  } else {
    # Not in tmux or not configured for fzf-tmux, use fzf directly
    # Add --height option for plain fzf
    let final_fzf_opts = ['--height', $height_opt] | append $fzf_prefinal_opt

    if $has_walker or ($stdin_content == null) {
      # Run directly if walker or no stdin provided
      fzf ...$final_fzf_opts
    } else {
      # Pipe captured stdin to fzf
      $stdin_content | fzf ...$final_fzf_opts
    }
  }
}

# Generate host list for ssh/telnet
def __fzf_list_hosts_nu [] {
  # Translate the Zsh pipeline using Nu commands and external tools
  let ssh_configs = try { open ~/.ssh/config | lines } catch { [] }
  let ssh_configs_d = try { open ~/.ssh/config.d/* | lines } catch { [] }
  let ssh_config_global = try { open /etc/ssh/ssh_config | lines } catch { [] }
  let known_hosts = try { open ~/.ssh/known_hosts | lines } catch { [] }
  let hosts_file = try { open /etc/hosts | lines } catch { [] }

  [
    (
      # Process ssh config files
      $ssh_configs | append $ssh_configs_d | append $ssh_config_global
      | where {|it| ($it | str downcase | str starts-with 'host') or ($it | str downcase | str starts-with 'hostname') }
      | parse --regex '^\s*host(?:name)?\s+(?<hosts>.+)' # Extract hosts after keyword
      | default { hosts: null } # Handle lines that don't match regex
      | get hosts
      | where {|it| $it != null }
      | split row ' '
      | where {|it| not ($it =~ '[*?%]') } # Exclude patterns containing *, ?, or %
    )
    (
      # Process known_hosts file
      $known_hosts
      | parse --regex '^(?:\[)?(?<hosts>[a-z0-9.,:_-]+)' # Extract hostnames (possibly in [], possibly comma-separated) - added underscore
      | default { hosts: null }
      | get hosts
      | where {|it| $it != null }
      | each { |it| $it | split row ',' } # Split comma-separated hosts if any
      | flatten
    )
    (
      # Process /etc/hosts file
      $hosts_file
      | where { |it| not ($it | str starts-with '#') } # Ignore comments
      | where { |it| not ($it | str trim | is-empty) } # Ignore empty lines
      | where { |it| not ($it | str contains '0.0.0.0') } # Ignore 0.0.0.0
      | str replace --regex '#.*$' '' # Remove trailing comments
      | parse --regex '^\s*\S+\s+(?<hosts>.+)' # Extract hosts part (after IP)
      | default { hosts: null }
      | get hosts
      | where {|it| $it != null }
      | split row ' ' # Split multiple hosts on the same line
    )
  ]
  | flatten # Combine all lists into a single stream
  | where {|it| not ($it | is-empty) } # Remove empty entries
  | sort | uniq # Sort and remove duplicates
}


# Base function for path/directory completion
def __fzf_generic_path_completion_nu [
    prefix: string,           # The text before the trigger
    compgen_cmd_name: string,        # not used
    fzf_opts_arg: list<string>, # Extra options for fzf
    suffix: string           # Suffix to add to selection (e.g., "/")
] {
  # --- Determine walker root and initial query from the raw prefix ---
  let raw_prefix = $prefix # Use the original prefix before any expansion

  mut walker_root = "."
  mut initial_query = ""

  if ($raw_prefix | is-empty) {
      # Case: "**"
      $walker_root = "."
      $initial_query = ""
  } else if ($raw_prefix | str contains (char separator)) {
      # Case: "dir/subdir/partial**" or "dir/**"
      $walker_root = $raw_prefix | path dirname
      $initial_query = $raw_prefix | path basename
      # Handle edge case where prefix ends with separator, e.g., "dir/"
      if ($raw_prefix | str ends-with (char separator)) {
          # Remove trailing separator to get the intended directory
          $walker_root = $raw_prefix | str substring 0..-2
          $initial_query = ""
      }
      # Ensure walker_root isn't empty if prefix was like "/file**"
      # or if path dirname returned empty string for some reason (e.g. prefix="file/")
      if ($walker_root | is-empty) {
        if ($raw_prefix | str starts-with (char separator)) {
          $walker_root = (char separator) 
        } else if ($raw_prefix | str ends-with (char separator)) {
          $walker_root = $raw_prefix | str substring 0..-2 
        } else { $walker_root = "." } # Fallback if dirname weirdly fails
      }

  } else {
      # Case: "partial**" (no slashes)
      $walker_root = "."
      $initial_query = $raw_prefix
  }

  # --- Candidate Generation ---
  # Keep existing logic for custom generators vs walker, but use newly calculated values.
  # Custom generators might still expect/need an absolute path. Expand walker_root only for them.
 
  # --- Prepare FZF options ---
  let completion_type_opts = if $suffix == '/' {
      $env.FZF_COMPLETION_DIR_OPTS? | default '' | split words
  } else {
      $env.FZF_COMPLETION_PATH_OPTS? | default '' | split words
  }

  let walker_type = if ($suffix == '/') {
      "dir,follow"
  } else {
      "file,dir,follow,hidden"
  }
  # Use the 'walker_root' calculated at the beginning
  let $fzf_all_opts = ["--scheme=path", "--walker", $walker_type, "--walker-root", $walker_root] 
    | append $fzf_opts_arg 
    | append $completion_type_opts

  # Call FZF run
  let fzf_selection = ( __fzf_comprun_nu "fzf-path-completion-walker" $initial_query $fzf_all_opts ) | str trim
  

  # --- Format Selection ---
  # Reconstruct the full path relative to the original prefix structure,
  # as fzf walker output is relative to --walker-root.
  # let completed_item = if ($fzf_selection | is-not-empty) {
  #     let joined_path = if ($fzf_selection | path type) == 'absolute' or $walker_root == '.' {
  #         # If selection is absolute OR walker_root was '.', use selection as is.
  #         $fzf_selection
  #     } else {
  #         # Otherwise, join the walker_root and the selection.
  #         $walker_root | path join $fzf_selection
  #     }
  #     # Add suffix (e.g., "/" for directories)
  #     $joined_path + $suffix
  # } else {
  #     "" # No selection
  # }

  let completed_item = $fzf_selection


  # --- Return Result ---
  if ($completed_item | is-not-empty) {
      [$completed_item]
  } else {
      []
  }
}

# Specific path completion wrapper
def _fzf_path_completion_nu [prefix: string] {
  # Zsh args: base, lbuf, _fzf_compgen_path, "-m", "", " "
  # Nu: prefix, empty command name (use find), ["-m"], "", " "
  __fzf_generic_path_completion_nu $prefix "" ["-m"] ""
}

# General completion helper for commands that feed a list to fzf
# This is called by ssh, export, unalias, kill. everything exept path and dir
def _fzf_complete_nu [
    query: string,              # The initial query string for fzf
    data_gen_closure: closure,    # Closure that generates candidates
    fzf_opts_arg: list<string>,  # Extra options for fzf (like -m, +m)
    post_process_closure: closure # Closure to process the selected item (optional)
] {
  
  # Generate candidates using the provided command
  let candidates = try {
    do $data_gen_closure
  } catch {
    # Capture the actual error object provided by the catch block
    let actual_error = $in
    # Print a more informative error message including the actual error details
    print -e $"Error executing data_gen closure. Closure code: ($data_gen_closure). Actual error: ($actual_error)"
    []
  }

  # Run fzf and get selection
  let fzf_selection = $candidates | to text 
  | __fzf_comprun_nu "fzf-helper" $query $fzf_opts_arg 
  | str trim # Trim potential trailing newline from fzf

  # Apply post-processing if closure provided and selection is not empty
  let processed_selection = if ($fzf_selection | is-not-empty) and ($post_process_closure != null) {
    # Call the post-processing closure with the selection
    try {
      do $post_process_closure $fzf_selection
    } catch {
      print -e $"Error executing post_process closure: ($post_process_closure)"
      $fzf_selection # Return original selection on error
    }
  } else {
    $fzf_selection
  }

  if not ($processed_selection | is-empty) {
    [$processed_selection] # Return as list
  } else {
    []
  }
}

# SSH/Telnet completion
def _fzf_complete_ssh_nu [prefix: string, input_line_before_trigger: string] {
  let words = ($input_line_before_trigger | split row ' ')
  let word_count = $words | length

  # Find the index of the word being completed (which is the prefix)
  # If prefix is empty, completion happens after a space, index is word_count
  # If prefix is not empty, it's the last word, index is word_count - 1
  let completion_index = if ($prefix | is-empty) { $word_count } else { $word_count - 1 }

  mut handled = false
  mut completion_result = [] # List of completion strings to return

  # Check for -i, -F, -E flags immediately preceding the cursor position
  if $completion_index > 0 {
    let prev_arg = ($words | get ($completion_index - 1))
    if ($prev_arg in ['-i', '-F', '-E']) {
      $handled = true
      # Call path completion with the current prefix
      $completion_result = (_fzf_path_completion_nu $prefix)
    }
  }

  # If not handled by path completion, do host completion
  if not $handled {
    let user_part = if ($prefix | str contains "@") { ($prefix | split column "@" | first) + "@" } else { "" }
    # The part after '@' (or the whole prefix if no '@') is the initial query for fzf
    let query = if ($prefix | str contains "@") { $prefix | split column "@" | get 1 } else { $prefix }

    let host_candidates_gen = {||
      __fzf_list_hosts_nu
      | each {|host_item| $user_part + $host_item } # Prepend user@ if present in prefix
    }

    # Zsh options: +m -- ; Nu: pass ["+m"]
    # Pass the host part of the prefix to _fzf_complete_nu for the initial query
    let selected_host = (_fzf_complete_nu $query $host_candidates_gen ["+m"] {}) # Pass host_prefix here
    if not ($selected_host | is-empty) {
      $completion_result = $selected_host # _fzf_complete_nu returns a list
    }
  }

  $completion_result
}

# Export completion
def _fzf_complete_export_nu [query: string] {
  let vars_gen_closure = {|| env | get name } # Nushell `env` provides names directly
  # Zsh options: -m -- ; Nu: pass ["-m"] ; +m = multiple choice
  _fzf_complete_nu $query $vars_gen_closure ["-m"] {}
}

# Unset completion (same as export)
def _fzf_complete_unset_nu [query: string] {
  _fzf_complete_export_nu $query # Re-use export logic
}

# Unalias completion
def _fzf_complete_unalias_nu [query: string] {
  let aliases_gen_closure = {|| aliases | get alias } # Use 'alias' column from `aliases` command
  # Zsh options: +m -- ; Nu: pass ["+m"] ; +m = multiple choice
  _fzf_complete_nu $query $aliases_gen_closure ["+m"] {}
}

# Kill completion post-processor (extracts PID)
def _fzf_complete_kill_post_get_pid [selected_line: string] {
  # Assuming standard ps output where PID is the second column
  $selected_line | from ssv --noheaders | get 0.column1
}

# Kill completion to get process PID
def _fzf_complete_kill_nu [query: string] {
  let ps_gen_closure = {|| # Define ps generator as a closure
    # Try standard ps, then busybox, then cygwin format approximation
    # Use `^ps` to ensure external command execution
    try {
      ^ps -eo user,pid,ppid,start,time,command | lines # Keep header for --header-lines=1
    } catch {
      try {
        ^ps -eo user,pid,ppid,time,args | lines # BusyBox?
      } catch {
        try {
          ^ps --everyone --full --windows | lines # Cygwin?
        } catch {
          print -e "Error: ps command failed."
          [] # Return empty list on failure
        }
      }
    }
  }

  # Note: Complex Zsh FZF bindings for kill (click-header transformer) are omitted for simplicity.
  # Users can set custom bindings via FZF_DEFAULT_OPTS if needed.
  let kill_post_closure = {|selected_line| _fzf_complete_kill_post_get_pid $selected_line }

  let fzf_opts = ["-m", "--header-lines=1", "--no-preview", "--wrap", "--color", "fg:dim,nth:regular"]

  _fzf_complete_nu $query $ps_gen_closure $fzf_opts $kill_post_closure
}


# --- Main FZF Completion Handler (bound to Tab) ---

let fzf_external_completer = {|current_line, cursor_pos|



  let trigger: string = $env.FZF_COMPLETION_TRIGGER? | default '**'
  if ($trigger | is-empty) { return null } # Cannot work with a trigger set to '' (blank)



  let line_before_cursor: string = $current_line | str substring 0..<$cursor_pos

  if ($line_before_cursor | str ends-with $trigger) {
    # --- Trigger Found ---

    # Store the line content just before the trigger for context
    let length_without_trigger: int = ($line_before_cursor | str length) - ($trigger | str length)
    let line_without_trigger: string = $line_before_cursor | str substring 0..<$length_without_trigger

    # Identify command word (first word) and the prefix being completed
    let spans = ($line_without_trigger | split row ' ')
    let cmd_word = ($spans | first | default "")

    # Calculate the prefix (part before the trigger in the last span)
    let prefix = if ($line_without_trigger | str ends-with " ") {
        ""
    } else {
        $spans | last | default ""
    }

    # Calculate the start position of the prefix within the original line
    let start_replace_pos = ($line_without_trigger | str length) - ($prefix | str length)
    # The end position of the replacement is the cursor position (end of the trigger)
    let end_replace_pos = $cursor_pos


    # --- Dispatch to Completer ---
    mut completion_results = [] # Will hold the list of strings from the completer

    match $cmd_word {
        "ssh" | "scp" | "sftp" | "telnet" => { $completion_results = (_fzf_complete_ssh_nu $prefix $line_without_trigger) }
        "export" | "printenv" => { $completion_results = (_fzf_complete_export_nu $prefix) }
        "unset" => { $completion_results = (_fzf_complete_unset_nu $prefix) }
        "unalias" => { $completion_results = (_fzf_complete_unalias_nu $prefix) }
        "kill" => { $completion_results = (_fzf_complete_kill_nu $prefix) }
        "cd" | "pushd" | "rmdir" => { $completion_results = (__fzf_generic_path_completion_nu $prefix "" [] "/") }
        # Add other command-specific completions here
        _ => {
            # Default to path completion if no specific command matches or cmd_word is empty
            $completion_results = (_fzf_path_completion_nu $prefix)
        }
    }

    

    if not ($completion_results | is-empty) {
      # Currently, assumes only one completion item is selected and returned.
      # FZF multi-select (-m) handled by path/kill completers returning list.
      # If multiple items are returned, how to insert? For now, take the first.
      let selected_completion = $completion_results | first

      # Add a space after the completion unless it's a directory ending with a slash
      let text_to_insert = if ($selected_completion | str ends-with "/") {
        $selected_completion # Directories already have trailing slash from completer
      } else {
        $selected_completion + " " # Files/others get a space added
      }

      # Construct the new line buffer string:
      # Part before the prefix + selected completion + part after the original cursor
      let part_before_replace = $current_line | str substring 0..($start_replace_pos - 1)
      let part_after_replace = $current_line | str substring $cursor_pos..-1
      let new_buffer_string = $part_before_replace + $selected_completion + $part_after_replace

      # Replace the entire buffer with the new string
      # {{change 1}} - Use commandline edit --replace to replace the whole buffer
      [{value: $new_buffer_string }] 

      # Calculate and set the new cursor position
      # It should be at the end of the inserted text
      # let new_cursor_pos = $start_replace_pos + ($text_to_insert | str length)
      # commandline set-cursor $new_cursor_pos
    } else {
      # No completion selected/found. Just remove the trigger.

      # Calculate the start position of the trigger within the original line
      let start_trigger_pos = $cursor_pos - ($trigger | str length)

      # Construct the new line buffer string:
      # Part before the trigger + part after the original cursor
      let part_before_trigger = $current_line | str substring 0..$start_trigger_pos
      let part_after_trigger = $current_line | str substring $cursor_pos..
      let new_buffer_string = $part_before_trigger + $part_after_trigger

      # Replace the entire buffer with the new string
      [{value: $new_buffer_string }] 

      # Set the cursor position to the end of the text that was before the trigger
      #commandline set-cursor $start_trigger_pos
    }

  } else {
    # --- Trigger Not Found ---
    
    # Pass the same original value
    # Nushell currently doesn't have an fallback option
    [{value: $current_line}]

    # There isn't a way to programatically call the default completer.
  }
}

# --- Register the Tab key binding ---



# This replaces the default Tab behavior with our fzf handler.
# See the warning in `fzf_tab_handler` regarding default completion fallback.
export-env {
  # Prevent adding the keybinding multiple times if the script is sourced again
  if not ($env.__fzf_completion_keybinding_loaded? | default false) {
    $env.__fzf_completion_keybinding_loaded = true

    # 1. Define the FZF-style completions menu
    let fzf_trigger_menu_config = {
        name: "fzf_trigger_completions" # Unique name for this menu
        marker: "ft> "                 # Marker when this menu's source is active
        only_buffer_difference: false  # Source function expects full line before cursor
        type: {
            layout: columnar          # Nushell's list-style menu
            page_size: 10         # How many items per page
        }
        style: { # Standard Nushell menu styling
            text: green
            selected_text: green_reverse
            description_text: yellow
        }
        source: $fzf_external_completer
            } # The function that generates completions
    

    # Add this menu to $env.config.menus
    $env.config = ($env.config | upsert menus (($env.config.menus? | default []) | append $fzf_trigger_menu_config))


    let updated = (
      $env.config.keybindings
      | each {|it|
          if $it.name == "completion_menu" {
            let new_event = (
              try {
                let original_until = $it.event.until
                $it.event
                | upsert until ([{ send: menu name: "fzf_trigger_completions" }] ++ $original_until)
              } catch {|err|
                # If something goes wrong (e.g. no 'until'), return the original event untouched
                $it.event
              }
            )
            $it | upsert event $new_event
          } else {
            $it
          }
        }
    )
    
    $env.config = ($env.config | upsert keybindings $updated)

  }
}
