function __fzf_cd_widget -d "Change directory"
  set -q FZF_ALT_C_COMMAND; or set -l FZF_ALT_C_COMMAND "
  command find -L . \\( -path '*/\\.*' -o -fstype 'devfs' -o -fstype 'devtmpfs' \\) -prune \
  -o -type d -print 2> /dev/null | sed 1d | cut -b3-"
  eval "$FZF_ALT_C_COMMAND | "(__fzfcmd)" +m --select-1 --exit-0 $FZF_ALT_C_OPTS" | read -l result
	[ "$result" ]; and cd $result
  commandline -f repaint
end
