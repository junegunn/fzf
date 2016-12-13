# Use last token $cwd_esc as root for the 'find' command
function __fzf_file_widget -d "List files and folders"
	set -l cwd (commandline -t)
	## The commandline token might be escaped, we need to unescape it.
	set cwd (eval "printf '%s' $cwd")
	if [ ! -d "$cwd" ]
		set cwd .
	end

	set -q FZF_CTRL_T_COMMAND; or set -l FZF_CTRL_T_COMMAND "
	command find -L \$cwd \\( -path \$cwd'*/\\.*' -o -fstype 'devfs' -o -fstype 'devtmpfs' \\) -prune \
	-o -type f -print \
	-o -type d -print \
	-o -type l -print 2> /dev/null | sed 1d"

	eval "$FZF_CTRL_T_COMMAND | "(__fzfcmd)" -m $FZF_CTRL_T_OPTS" | while read -l r; set result $result $r; end
	if [ -z "$result" ]
		commandline -f repaint
		return
	end

	if [ "$cwd" != . ]
		## Remove last token from commandline.
		commandline -t ""
	end
	for i in $result
		commandline -it -- (string escape $i)
		commandline -it -- ' '
	end
	commandline -f repaint
end
