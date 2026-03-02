#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ completion-external.nu


# An implementation of completion.nu
# This loads FZF as an Nushell External Completer
# https://www.nushell.sh/cookbook/external_completers.html

# It's the most stable implementation.
# The drawback is that it does't work for completing some commands, like 'cd' and 'ls' on Nushell >= 0.103.0
# https://www.nushell.sh/blog/2025-03-18-nushell_0_103_0.html#external-completers-are-no-longer-used-for-internal-commands-toc


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
  let default_opts      = $env.FZF_DEFAULT_OPTS? | default ''
  let default_opts_file = $env.FZF_DEFAULT_OPTS_FILE? | default ''

  let file_opts = try {
     open $default_opts_file | lines | str trim | where not ($in | is-empty)
  } catch {
     [] # Return empty list on error (e.g., file not found)
  }

  # Build options list
                                                                                                     # Start with the prepend argument
  return $prepend | append $file_opts                                                                # Append options from file
                  | append ($default_opts | split words | where not ($in | is-empty))                # Append options from $FZF_DEFAULT_OPTS
                  | append ($append | split words | where not ($in | is-empty))                      # Append options from function argument
                  | where {|it| try { ($it | is-string) and not ($it | is-empty) } catch { false } } # Filter to keep only non-empty strings, safely handling potential errors
}

# Wrapper for running fzf or fzf-tmux
def __fzf_comprun_nu [ context_name: string       # e.g., "fzf-completion" , "fzf-helper" - mainly for potential debugging
                     , query:        string       # The initial query string for fzf
                     , fzf_opts_arg: list<string> # Remaining options for fzf/fzf-tmux
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
  let ssh_configs       = try { open ~/.ssh/config       | lines } catch { [] }
  let ssh_configs_d     = try { open ~/.ssh/config.d/*   | lines } catch { [] }
  let ssh_config_global = try { open /etc/ssh/ssh_config | lines } catch { [] }
  let known_hosts       = try { open ~/.ssh/known_hosts  | lines } catch { [] }
  let hosts_file        = try { open /etc/hosts          | lines } catch { [] }

  [
    (
      # Process ssh config files
      $ssh_configs | append $ssh_configs_d | append $ssh_config_global
                   | where {|it| ($it | str downcase | str starts-with 'host') or ($it | str downcase | str starts-with 'hostname') }
                   | parse --regex '^\s*host(?:name)?\s+(?<hosts>.+)' # Extract hosts after keyword
                   | default { hosts: null }                          # Handle lines that don't match regex
                   | get hosts
                   | where {|it| $it != null }
                   | split row ' '
                   | where {|it| not ($it =~ '[*?%]') }               # Exclude patterns containing *, ?, or %
    )
    (
      # Process known_hosts file
      $known_hosts | parse --regex '^(?:\[)?(?<hosts>[a-z0-9.,:_-]+)' # Extract hostnames (possibly in [], possibly comma-separated) - added underscore
                   | default { hosts: null }
                   | get hosts
                   | where {|it| $it != null }
                   | each { |it| $it | split row ',' }                # Split comma-separated hosts if any
                   | flatten
    )
    (
      # Process /etc/hosts file
      $hosts_file | where { |it| not ($it | str starts-with '#') }    # Ignore comments
                  | where { |it| not ($it | str trim | is-empty) }    # Ignore empty lines
                  | where { |it| not ($it | str contains '0.0.0.0') } # Ignore 0.0.0.0
                  | str replace --regex '#.*$' ''                     # Remove trailing comments
                  | parse --regex '^\s*\S+\s+(?<hosts>.+)'            # Extract hosts part (after IP)
                  | default { hosts: null }
                  | get hosts
                  | where {|it| $it != null }
                  | split row ' '                                     # Split multiple hosts on the same line
    )
  ]
  | flatten # Combine all lists into a single stream
  | where {|it| not ($it | is-empty) } # Remove empty entries
  | sort | uniq # Sort and remove duplicates
}


# Base function for path/directory completion
def __fzf_generic_path_completion_nu [ prefix:           string       # The text before the trigger
                                     , compgen_cmd_name: string       # not used
                                     , fzf_opts_arg:     list<string> # Extra options for fzf
                                     , suffix:           string       # Suffix to add to selection (e.g. , "/")
                                     ] {
  # --- Determine walker root and initial query from the raw prefix ---
  let raw_prefix = $prefix # Use the original prefix before any expansion

  mut walker_root   = "."
  mut initial_query = ""

  if ($raw_prefix | is-empty) {
    # Case: "**"
    $walker_root   = "."
    $initial_query = ""
  } else if ($raw_prefix | str contains (char separator)) {
    # Case: "dir/subdir/partial**" or "dir/**"
    $walker_root   = $raw_prefix | path dirname
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
    $walker_root   = "."
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
  let $fzf_all_opts = ["--scheme=path", "--walker", $walker_type, "--walker-root", $walker_root] | append $fzf_opts_arg
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
def _fzf_complete_nu [ query:                  string       # The initial query string for fzf
                     , data_gen_closure:       closure      # Closure that generates candidates
                     , fzf_opts_arg:           list<string> # Extra options for fzf (like -m, +m)
                     , --post_process_closure: closure      # Closure to process the selected item (optional)
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
  let processed_selection = if ($fzf_selection | is-not-empty) and ($post_process_closure | is-not-empty) {
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
    [($processed_selection | lines | str join ' ')]
  } else {
    []
  }
}

# SSH/Telnet completion
def _fzf_complete_ssh_nu [ prefix:                    string
                         , input_line_before_trigger: string
                         ] {
  let words      = ($input_line_before_trigger | split row ' ')
  let word_count = $words | length

  # Find the index of the word being completed (which is the prefix)
  # If prefix is empty, completion happens after a space, index is word_count
  # If prefix is not empty, it's the last word, index is word_count - 1
  let completion_index = if ($prefix | is-empty) { $word_count } else { $word_count - 1 }

  mut handled           = false
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
    let selected_host = (_fzf_complete_nu $query $host_candidates_gen ["+m"]) # Pass host_prefix here
    if not ($selected_host | is-empty) {
      $completion_result = $selected_host # _fzf_complete_nu returns a list
    }
  }

  $completion_result
}

def _fzf_list_pacman_packages [--installed] {
  let pkg_line_regex        = '^[^/ ]+/(\S+).*$'
  let accumulating_closure  = { |line, acc|
    match $line {
      $l if $l =~ $pkg_line_regex => ( $acc | append ($l | str replace -r $pkg_line_regex '${1}') )
      _                           => (                                                            )
    }
  }
  do (if $installed {{|| pacman -Qs . }} else {{|| pacman -Ss .}}) | lines | where $it =~ $pkg_line_regex | each {$in | str replace -r $pkg_line_regex '${1}'}
}

def _fzf_complete_pacman_nu [ prefix:                    string
                            , input_line_before_trigger: string
                            ] {
  let command_words = $input_line_before_trigger | split row ' '
  let sub_command   = $command_words | skip 1 | first
  match $sub_command {
    $s if $s =~ "-S[bcdgilpqrsuvwy]*"     => ( _fzf_complete_nu $prefix {_fzf_list_pacman_packages            } ["-m"] )
    $s if $s =~ "-Q[bcdegiklmnpqrstuv]*"  => ( _fzf_complete_nu $prefix {_fzf_list_pacman_packages --installed} ["-m"] )
    $s if $s =~ "-F[blqrvxy]*"            => ( _fzf_complete_nu $prefix {_fzf_list_pacman_packages            } ["-m"] )
    $s if $s =~ "-R[bcdnprsuv]*"          => ( _fzf_complete_nu $prefix {_fzf_list_pacman_packages --installed} ["-m"] )
    _                                     => (                                                                         )
  }
}

def _fzf_complete_pass_nu [prefix: string] {
  let passwordstore_files_gen_closure = {||
    ls ~/.password-store/**/*.gpg | get name | each {$in | str replace -r '^.*?\.password-store/(.*).gpg' '${1}' }
  }
  _fzf_complete_nu $prefix $passwordstore_files_gen_closure ["+m"]
}

# Export completion
def _fzf_complete_export_nu [query: string] {
  let vars_gen_closure = {|| env | get name } # Nushell `env` provides names directly
  # Zsh options: -m -- ; Nu: pass ["-m"] ; +m = multiple choice
  _fzf_complete_nu $query $vars_gen_closure ["-m"]
}

# Unset completion (same as export)
def _fzf_complete_unset_nu [query: string] {
  _fzf_complete_export_nu $query # Re-use export logic
}

# Unalias completion
def _fzf_complete_unalias_nu [query: string] {
  let aliases_gen_closure = {|| aliases | get alias } # Use 'alias' column from `aliases` command
  # Zsh options: +m -- ; Nu: pass ["+m"] ; +m = multiple choice
  _fzf_complete_nu $query $aliases_gen_closure ["+m"]
}

# Kill completion post-processor (extracts PID)
def _fzf_complete_kill_post_get_pid [selected_line: string] {
  # Assuming standard ps output where PID is the second column
  $selected_line | lines | each { $in | from ssv --noheaders | get 0.column1 } | to text
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

  _fzf_complete_nu $query $ps_gen_closure $fzf_opts --post_process_closure $kill_post_closure
}


# --- Main FZF External Completer ---

# This function is registered with Nushell's external completion system.
# It gets called when Tab is pressed.
let fzf_external_completer = {|spans|
  let trigger: string = $env.FZF_COMPLETION_TRIGGER? | default '**'

  if ($trigger | is-empty)     { return null } # Cannot work with empty trigger
  if (($spans | length ) == 0) { return null } # Nothing to complete

  let last_span = $spans | last
  let line_before_cursor = $spans | str join ' ' # Reconstruct line for context

  if ($last_span | str ends-with $trigger) {
    # --- Trigger Found ---

    let cmd_word = ($spans | first | default "")

    # Calculate the prefix (part before the trigger in the last span)
    let prefix = $last_span | str substring 0..(-1 * ($trigger | str length) - 1)

    # Reconstruct the line content *before* the trigger for context
    # This is an approximation based on spans
    let line_without_trigger = $spans | take (($spans | length) - 1) | append $prefix | str join ' '

    # --- Dispatch to Completer ---
    mut completion_results = [] # Will hold the list of strings from the completer

    match $cmd_word {
      "pacman"                          => { $completion_results = (_fzf_complete_pacman_nu $prefix $line_without_trigger) }
      "pass"                            => { $completion_results = (_fzf_complete_pass_nu $prefix)                         }
      "ssh" | "scp" | "sftp" | "telnet" => { $completion_results = (_fzf_complete_ssh_nu $prefix $line_without_trigger)    }
      # "export" | "printenv"             => { $completion_results = (_fzf_complete_export_nu $prefix)                    }
      # "unset"                           => { $completion_results = (_fzf_complete_unset_nu $prefix)                     }
      # "unalias"                         => { $completion_results = (_fzf_complete_unalias_nu $prefix)                   }
      "kill"                            => { $completion_results = (_fzf_complete_kill_nu $prefix)                         }
      "cd" | "pushd" | "rmdir"          => { $completion_results = (__fzf_generic_path_completion_nu $prefix "" [] "/")    }
      # Add other command-specific completions here
      _                                 => {
        # Default to path completion if no specific command matches or cmd_word is empty
        $completion_results = (_fzf_path_completion_nu $prefix)
      }
    }

    # --- Return Results ---
    # The _fzf_... functions return a list of completion strings.
    # Nushell's completer expects the suggestions for the token being completed (prefix + trigger).
    # The results from the helper functions should be the final desired strings.
    # We don't need to manually add spaces; Nushell handles that.
    $completion_results # Return the list directly
  } else {
    # --- Trigger Not Found ---
    # Return null to let Nushell fall back to other completers (e.g., default file completion).
    null
  }
}

# --- WRAPPER AND REGISTRATION ---

# Get the currently configured external completer, if any exists
let previous_external_completer = $env.config? | get completions? | get external? | get completer?

# Define the new wrapper completer
let fzf_wrapper_completer = {|spans|
  # 1. Try the FZF completer logic first
  let fzf_result = do $fzf_external_completer $spans

  # 2. If FZF returned a result (a list, even an empty one), return it.
  #    `null` means FZF didn't handle it because the trigger wasn't present.
  if $fzf_result != null {
    $fzf_result
  } else {
    # 3. FZF didn't handle it, so call the previous completer (if it exists).
    if $previous_external_completer != null {
      do $previous_external_completer $spans
    } else {
      # 4. No previous completer, and FZF didn't handle it. Return null.
      null
    }
  }
}

# Register the new wrapper completer
# This ensures external completions are enabled and sets our wrapper.
$env.config = $env.config | upsert completions {
  external: {
    enable: true
    completer: $fzf_wrapper_completer
  }
}

#  vim: set sts=2 ts=2 sw=2 tw=120 et :

