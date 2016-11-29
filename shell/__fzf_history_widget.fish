function __fzf_history_widget -d "Show command history"
	history | eval (__fzfcmd) +m --tiebreak=index $FZF_CTRL_R_OPTS -q '(commandline)' | read -l result
	and commandline -- $result
	commandline -f repaint
end
