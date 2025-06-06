#!/bin/sh

# This script applies the contents of "common.sh" to the other files.

set -e

# Go to the directory that contains this script
dir=${0%"${0##*/}"}
if [ -n "$dir" ]; then
  cd "$dir"
fi

update() {
  {
    sed -n '1,/^#----BEGIN INCLUDE common\.sh/p' "$1"
    cat <<EOF
# NOTE: Do not directly edit this section, which is copied from "common.sh".
# To modify it, one can edit "common.sh" and run "./update-common.sh" to apply
# the changes. See code comments in "common.sh" for the implementation details.
EOF
    grep -v '^[[:blank:]]*#' common.sh # remove code comments in common.sh
    sed -n '/^#----END INCLUDE/,$p' "$1"
  } > "$1.part"

  mv -f "$1.part" "$1"
}

update completion.bash
update completion.zsh
update key-bindings.bash
update key-bindings.zsh
