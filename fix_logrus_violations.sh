#!/usr/bin/env bash
set -euo pipefail

readonly CONTAINING_DIR=$(unset CDPATH && cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
cd "$CONTAINING_DIR"



SEARCH_PATHS=(
    "$HOME"/go/pkg/mod/github.com/docker
    "$HOME"/go/pkg/mod/github.com/opencontainers
    "$HOME/go/pkg/mod/github.com/!microsoft"
)
FILES_STR=$(find "${SEARCH_PATHS[@]}" -type f -name '*.go')
IFS=$'\n' read -rd '' -a FILES <<< "$FILES_STR" || true
for FILE in "${FILES[@]}"; do
    if grep -q -w 'github.com/Sirupsen/logrus' "$FILE"; then
        echo "$FILE"
        sed -i '' -e 's:github.com/Sirupsen/logrus:github.com/sirupsen/logrus:g' "$FILE"
    fi
done
