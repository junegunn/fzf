# Unset fzf variables
set -e FZF_DEFAULT_COMMAND FZF_DEFAULT_OPTS FZF_DEFAULT_OPTS_FILE FZF_TMUX FZF_TMUX_OPTS
set -e FZF_CTRL_T_COMMAND FZF_CTRL_T_OPTS FZF_ALT_C_COMMAND FZF_ALT_C_OPTS FZF_CTRL_R_OPTS
set -e FZF_API_KEY
# Unset completion-specific variables
set -e FZF_COMPLETION_TRIGGER FZF_COMPLETION_OPTS

set -gx FZF_DEFAULT_OPTS "--no-scrollbar --pointer '>' --marker '>'"
set -gx FZF_COMPLETION_TRIGGER '++'
set -gx fish_history fzf_test

# Add fzf to PATH
fish_add_path <%= BASE %>/bin

# Source key bindings and completion
source "<%= BASE %>/shell/key-bindings.fish"
source "<%= BASE %>/shell/completion.fish"
