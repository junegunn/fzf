## get the real path of plugin
# fzf_dir="${BASH_SOURCE[0]}" in zsh is equivalent to below
fzf_dir="${(%):-%N}"
# resolve $SOURCE until the file is no longer a symlink
while [ -L "$fzf_dir" ]; do
  APP_PATH="$( cd -P "$( dirname "$fzf_dir" )" && pwd )"
  fzf_dir="$(readlink "$fzf_dir")"
  # if $fzf_dir was a relative symlink, we need to resolve it relative to the path
  # where the symlink file was located
  [[ $fzf_dir != /* ]] && fzf_dir="$APP_PATH/$fzf_dir"
done
fzf_path="$( cd -P "$( dirname "$fzf_dir" )" && pwd )"

# only enable plugins when fzf and fzf-tmux are installed correctly
if [ -x $fzf_path/bin/fzf ] && [ -x $fzf_path/bin/fzf-tmux ]; then

  # export $PAHT and $MANPATH
  export PATH="$PATH:$fzf_path/bin"
  export MANPATH="$MANPATH:$fzf_path/man"

  # auto completion is broken with zsh-autosuggestion plugin
  # so comment out this line
  # export FZF_COMPLETION_TRIGGER='~~'
  # source "$fzf_path/shell/completion.zsh"

  source "$fzf_path/shell/key-bindings.zsh"
fi;
