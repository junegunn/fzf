#!/usr/bin/env bash

# This script applies the contents of "common.sh" to the other files.

set -e

dir=${0%"${0##*/}"}

update() {
  {
    sed -n "1,/^#----BEGIN INCLUDE $1/p" "$2"
    cat << EOF
# NOTE: Do not directly edit this section, which is copied from "$1".
# To modify it, one can edit "$1" and run "./update.sh" to apply
# the changes. See code comments in "$1" for the implementation details.
EOF
    echo
    grep -v '^[[:blank:]]*#' "$dir/$1" # remove code comments from the common file
    sed -n '/^#----END INCLUDE/,$p' "$2"
  } > "$2.part"

  mv -f "$2.part" "$2"
}

update "common.sh" "$dir/completion.bash"
update "common.sh" "$dir/completion.zsh"
update "common.sh" "$dir/key-bindings.bash"
update "common.sh" "$dir/key-bindings.zsh"
update "common.fish" "$dir/completion.fish"
update "common.fish" "$dir/key-bindings.fish"

# Check if --check is in ARGV
check=0
rest=()
for arg in "$@"; do
  case $arg in
    --check) check=1 ;;
    *) rest+=("$arg") ;;
  esac
done

fmt() {
  if ! grep -q "^#----BEGIN shfmt" "$1"; then
    if [[ $check == 1 ]]; then
      shfmt -d "$1"
      return $?
    else
      shfmt -w "$1"
    fi
  else
    {
      sed -n '1,/^#----BEGIN shfmt/p' "$1" | sed '$d'
      sed -n '/^#----BEGIN shfmt/,/^#----END shfmt/p' "$1" | shfmt --filename "$1"
      sed -n '/^#----END shfmt/,$p' "$1" | sed '1d'
    } > "$1.part"

    if [[ $check == 1 ]]; then
      diff -q "$1" "$1.part"
      ret=$?
      rm -f "$1.part"
      return $ret
    fi

    mv -f "$1.part" "$1"
  fi
}

for file in "${rest[@]}"; do
  fmt "$file" || exit $?
done
