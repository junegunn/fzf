__fzf_defaults() {
  # $1: Prepend to FZF_DEFAULT_OPTS_FILE and FZF_DEFAULT_OPTS
  # $2: Append to FZF_DEFAULT_OPTS_FILE and FZF_DEFAULT_OPTS
  printf '%s\n' "--height ${FZF_TMUX_HEIGHT:-40%} --min-height 20+ --bind=ctrl-z:ignore $1"
  command cat "${FZF_DEFAULT_OPTS_FILE-}" 2> /dev/null
  printf '%s\n' "${FZF_DEFAULT_OPTS-} $2"
}

__fzf_exec_awk() {
  # This function performs `exec awk "$@"` safely by working around awk
  # compatibility issues.
  #
  # To reduce an extra fork, this function performs "exec" so is expected to be
  # run as the last command in a subshell.
  if [[ -z ${__fzf_awk-} ]]; then
    __fzf_awk=awk
    if [[ $OSTYPE == solaris* && -x /usr/xpg4/bin/awk ]]; then
      # Note: Solaris awk at /usr/bin/awk is meant for backward compatibility
      # with an ancient implementation of 1977 awk in the original UNIX.  It
      # lacks many features of POSIX awk, so it is essentially useless in the
      # modern point of view.  To use a standard-conforming version in Solaris,
      # one needs to explicitly use /usr/xpg4/bin/awk.
      __fzf_awk=/usr/xpg4/bin/awk
    elif command -v mawk > /dev/null 2>&1; then
      # choose the faster mawk if: it's installed && build date >= 20230322 &&
      # version >= 1.3.4
      local n x y z d
      IFS=' .' read -r n x y z d <<< $(command mawk -W version 2> /dev/null)
      [[ $n == mawk ]] &&
        (((x * 1000 + y) * 1000 + z >= 1003004)) 2> /dev/null &&
        ((d >= 20230302)) 2> /dev/null &&
        __fzf_awk=mawk
    fi
  fi
  # Note: macOS awk has a quirk that it stops processing at all when it sees
  # any data not following UTF-8 in the input stream when the current LC_CTYPE
  # specifies the UTF-8 encoding.  To work around this quirk, one needs to
  # specify LC_ALL=C to change the current encoding to the plain one.
  LC_ALL=C exec "$__fzf_awk" "$@"
}
