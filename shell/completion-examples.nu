#     ____      ____
#    / __/___  / __/
#   / /_/_  / / /_
#  / __/ / /_/ __/
# /_/   /___/_/ completion-examples.nu
#
# Example custom completers for fzf's Nushell integration.
#
# To use these, add the desired entries to $env.FZF_COMPLETERS in your
# config.nu. Each closure receives two arguments:
#   - prefix: the text before the trigger (e.g. "vim" in "vim **<TAB>")
#   - spans:  the full command as a list of words (e.g. ["pacman", "-S", "vim**"])
#
# A closure can return either:
#   - a list of candidate strings (fzf will use default options), or
#   - a record with the following optional fields:
#       candidates: list<string>          # candidates to feed to fzf
#       opts:       list<string>          # custom fzf options (default: ["-m"])
#       post:       closure (|sel| ...)   # post-processing of the selected item
#
# Simple example:
#   $env.FZF_COMPLETERS = {
#       git: {|prefix, spans| ["branch-main", "branch-dev", "branch-feature"]}
#   }

# --- pacman / paru ---
# Completes package names for pacman and paru.
# Uses the spans to distinguish between subcommands:
#   -S (sync), -F (files): list available packages from repos
#   -Q (query), -R (remove): list installed packages
# Returns a record with custom fzf options for package preview.

$env.FZF_COMPLETERS = {}

$env.FZF_COMPLETERS.pacman = {|prefix, spans|
    let sub = $spans | skip 1 | first
    let candidates = (if ($sub =~ "-[SF]") {
        ^pacman -Slq | lines
    } else if ($sub =~ "-[QR]") {
        ^pacman -Qq | lines
    } else {
        []
    })
    {
        candidates: $candidates
        opts: ["-m", "--preview", "pacman -Si {}", "--prompt", "Package > "]
    }
}

$env.FZF_COMPLETERS.paru = {|prefix, spans|
    let sub = $spans | skip 1 | first
    let candidates = (if ($sub =~ "-[SF]") {
        ^pacman -Slq | lines
    } else if ($sub =~ "-[QR]") {
        ^pacman -Qq | lines
    } else {
        []
    })
    {
        candidates: $candidates
        opts: ["-m", "--preview", "pacman -Si {}", "--prompt", "Package > "]
    }
}

# --- pass (password-store) ---
# Completes entry names from ~/.password-store.
# Returns a simple list (no custom fzf options needed).

$env.FZF_COMPLETERS.pass = {|prefix, spans|
    try {
        ls ~/.password-store/**/*.gpg
        | get name
        | each {$in | str replace -r '^.*?\.password-store/(.*).gpg' '${1}'}
    } catch {
        []
    }
}

# --- Example with post-processing hook ---
# The "post" closure transforms the selected line after fzf returns.
# This is useful when the displayed line contains more information than
# what you want inserted on the command line (e.g. extracting a PID from
# a full "ps" output line).
#
# $env.FZF_COMPLETERS.mycommand = {|prefix, spans|
#     {
#         candidates: (^some-command | lines)
#         opts: ["+m", "--header-lines=1"]
#         post: {|selection| $selection | split row ' ' | get 0}
#     }
# }
