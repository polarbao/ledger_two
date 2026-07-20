#!/bin/sh

set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
verify_script="$script_dir/verify-task53-staging.sh"

sh -n "$verify_script"

umask_line="$(grep -n '^umask 077$' "$verify_script" | cut -d: -f1)"
mkdir_line="$(grep -n '^mkdir -p ' "$verify_script" | head -n 1 | cut -d: -f1)"

if [ -z "$umask_line" ]; then
  echo "verify-task53-staging.sh must set umask 077" >&2
  exit 1
fi
if [ -z "$mkdir_line" ] || [ "$umask_line" -ge "$mkdir_line" ]; then
  echo "umask 077 must be set before Task53 runtime artifacts are created" >&2
  exit 1
fi

echo "Task53 staging script security contract passed"
