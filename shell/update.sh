#!/usr/bin/env bash

# This script applies the contents of "common.sh" to the other files.

set -e

dir=${0%"${0##*/}"}

update() {
  {
    sed -n '1,/^#----BEGIN INCLUDE common\.sh/p' "$1"
    cat << EOF
# NOTE: Do not directly edit this section, which is copied from "common.sh".
# To modify it, one can edit "common.sh" and run "./update.sh" to apply
# the changes. See code comments in "common.sh" for the implementation details.
EOF
    echo
    grep -v '^[[:blank:]]*#' "$dir/common.sh" # remove code comments in common.sh
    sed -n '/^#----END INCLUDE/,$p' "$1"
  } > "$1.part"

  mv -f "$1.part" "$1"
}

update "$dir/completion.bash"
update "$dir/completion.zsh"
update "$dir/key-bindings.bash"
update "$dir/key-bindings.zsh"

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
