#!/usr/bin/env bash
#
# The purpose of this script is to demonstrate how to preview a file or an
# image in the preview window of fzf.
#
# Dependencies:
# - https://github.com/hpjansson/chafa
# - https://github.com/sharkdp/bat

file=$1
type=$(file --mime-type "$file")

if [[ ! $type =~ image/ ]]; then
  # Sometimes bat is installed as batcat.
  if command -v batcat > /dev/null; then
    batname="batcat"
  elif command -v bat > /dev/null; then
    batname="bat"
  else
    cat "$1"
    exit
  fi

  ${batname} --style="${BAT_STYLE:-numbers}" --color=always --pager=never -- "$file"
elif [[ $KITTY_WINDOW_ID ]]; then
  # 1. 'memory' is the fastest option but if you want the image to be scrollable,
  #    you have to use 'stream'.
  #
  # 2. The last line of the output is the ANSI reset code without newline.
  #    This confuses fzf and makes it render scroll offset indicator.
  #    So we remove the last line and append the reset code to its previous line.
  kitty icat --clear --transfer-mode=memory --stdin=no --place="${FZF_PREVIEW_COLUMNS}x$((FZF_PREVIEW_LINES))@0x0" "$file" | sed '$d' | sed $'$s/$/\e[m/'
elif command -v chafa > /dev/null; then
  chafa -f sixel -s "${FZF_PREVIEW_COLUMNS}x${FZF_PREVIEW_LINES}" "$file"
else
  file "$file"
fi
