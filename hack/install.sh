#!/usr/bin/env bash

set -euo pipefail

trap 'echo -ne "\e[0m"' EXIT

: "${BIN_DIR:=$HOME/.local/bin}"
BIN_NAME=cu
BIN_PATH=${BIN_DIR}/${BIN_NAME}

if [[ -z $SHELL ]]; then
    2>&1 echo "Could not detect your shell."
    exit 1
fi

comm=$(ps -o comm= -p $$ | sed 's#.*/##')
shell=$(echo "$SHELL" | sed 's#.*/##')

# Ensure we're in the same shell as the login shell
if [[ $comm != $shell ]]; then
    exec $SHELL --login -- $0 "$@"
    exit 1
fi

bashOrZsh=
if [[ $SHELL == */bash || $SHELL == */zsh ]]; then
    bashOrZsh=1
fi

if [[ $BIN_DIR == "$HOME/.local/bin" ]]; then
    mkdir -p "$BIN_DIR"
fi
if [[ ! -w "$BIN_DIR" ]]; then
    2>&1 echo "$BIN_DIR is not a writable directory"
	exit 1
fi

install -p "$BIN_NAME" "$BIN_DIR/"

echo "Installed $BIN_NAME binary to ${BIN_DIR}"

# Simulate an interactive environment by source-ing the user's rcfile.
rcfile="$HOME/.${shell}rc"
if [[ -f "$rcfile" ]]; then
    # do not abort if user's rcfile has errors
    set +e
    source "$rcfile"
    set -e
fi

if [[ $(command -v "$BIN_NAME") == $BIN_PATH ]]; then
    # The parent shell's cache may need to be updated.
    echo -e "You may need to execute the following to refresh your shell's command cache:\n"
    if [[ -n $bashOrZsh ]] then
        echo "  hash -r"
    else
        echo "  source \"$rcfile\""
    fi
    echo
    exit 0
fi

dir=$(command -v "$BIN_NAME" | xargs dirname)

if echo "$PATH" | tr : '\n' | grep -qF "${BIN_DIR}"; then
    warning="$dir is before $BIN_DIR in PATH\n"
fi

warning=${warning:-"PATH does not contain $BIN_DIR\n"}
echo -e "\e[38;2;200;100;100m\nYour shell is unable to locate the installed 'cu' binary\nbecause $warning\e[0m"
echo -e "Execute the following to configure your shell to locate 'cu' in $BIN_DIR:\n\e[33m"
echo "  echo 'export PATH=\"${BIN_DIR}:\$PATH\"' >> \"$rcfile\""
echo "  source \"$rcfile\""
echo -e "\e[0m"
