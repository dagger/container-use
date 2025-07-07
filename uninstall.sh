#!/bin/sh

# container-use uninstaller script

set -euo pipefail

main() {
    local BINARY_NAME="container-use"
    local INSTALL_DIR="${BIN_DIR:-$HOME/.local/bin}"
    local BINARY_PATH="$INSTALL_DIR/$BINARY_NAME"

    if [ ! -f "$BINARY_PATH" ]; then
        echo "container-use not found at $BINARY_PATH"
        exit 1
    fi

    # Safety check: don't delete from system paths or homebrew
    case "$BINARY_PATH" in
        /usr/bin/* | /bin/* | /usr/local/bin/* | /opt/homebrew/bin/*)
            echo "Error: Refusing to delete from system/brew path: $BINARY_PATH"
            echo "This script only removes container-use from user directories"
            exit 1
            ;;
    esac

    echo "Found container-use at: $BINARY_PATH"
    printf "Remove this file? (y/N): "
    read -r response

    case "$response" in
        [yY]|[yY][eE][sS])
            rm -f "$BINARY_PATH"
            echo "Removed $BINARY_PATH"

            # Also remove cu symlink if it exists
            local SYMLINK_PATH="$INSTALL_DIR/cu"
            if [ -L "$SYMLINK_PATH" ]; then
                rm -f "$SYMLINK_PATH"
                echo "Removed cu symlink at $SYMLINK_PATH"
            fi
            ;;
        *)
            echo "Cancelled"
            exit 1
            ;;
    esac
}

main "$@"
