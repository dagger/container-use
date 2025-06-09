#!/usr/bin/env bash

set -euo pipefail

: "${BIN_DIR:=$HOME/.local/bin}"

if [[ ! -f "$BIN_DIR/cu" ]]; then
    2>&1 echo "Nothing to uninstall at $BIN_DIR/cu"
	exit 1
fi

rm -f "${BIN_DIR}/cu"
echo "Uninstalled cu from $BIN_DIR"
