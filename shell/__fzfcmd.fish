function __fzfcmd
	set -q FZF_TMUX; or set FZF_TMUX 1
	if [ $FZF_TMUX -eq 1 ]
		if set -q FZF_TMUX_HEIGHT
			echo "fzf-tmux -d$FZF_TMUX_HEIGHT"
		else
			echo "fzf-tmux -d40%"
		end
	else
		echo "fzf"
	end
end
