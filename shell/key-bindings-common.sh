# Keep in sync with version for fish (in key-bindings.fish)
__fzf_history_dedup__() {
  if [ "$FZF_HIST_FIND_NO_DUPS" = on ]; then
    perl -ne 'print if !$seen{($_ =~ s/^[0-9\s]*//r)}++'
  else
    cat
  fi
}
