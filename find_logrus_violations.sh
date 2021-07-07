#!/usr/bin/env bash
set -euo pipefail

readonly CONTAINING_DIR=$(unset CDPATH && cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
cd "$CONTAINING_DIR"

FILES_STR=$(find "$HOME"/go -type f -name '*.go')
IFS=$'\n' read -rd '' -a FILES <<< "$FILES_STR" || true
for FILE in "${FILES[@]}"; do
    if grep -q -w 'github.com/sirupsen/logrus' "$FILE"; then
        echo "$FILE"
    fi
done

