#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

source "$REPO_ROOT/install.sh"

assert_equals() {
    local actual="$1"
    local expected="$2"
    local label="$3"

    if [ "$actual" != "$expected" ]; then
        echo "❌ $label"
        echo "  expected: $expected"
        echo "  actual:   $actual"
        exit 1
    fi
}

assert_contains() {
    local haystack="$1"
    local needle="$2"
    local label="$3"

    case "$haystack" in
        *"$needle"*)
            ;;
        *)
            echo "❌ $label"
            echo "  expected to find: $needle"
            exit 1
            ;;
    esac
}

DATA_HOME="$HOME/.local/share"
CONFIG_HOME="$HOME/.config"

assert_equals "$(completion_path linux bash container-use)" "$DATA_HOME/bash-completion/completions/container-use" "linux bash container-use"
assert_equals "$(completion_path linux zsh container-use)" "$DATA_HOME/zsh/site-functions/_container-use" "linux zsh container-use"
assert_equals "$(completion_path linux fish container-use)" "$CONFIG_HOME/fish/completions/container-use.fish" "linux fish container-use"
assert_equals "$(completion_path darwin bash container-use)" "/usr/local/etc/bash_completion.d/container-use" "darwin bash container-use"
assert_equals "$(completion_path darwin zsh container-use)" "/usr/local/share/zsh/site-functions/_container-use" "darwin zsh container-use"
assert_equals "$(completion_dir linux bash)" "$DATA_HOME/bash-completion/completions" "linux bash completion_dir"
assert_equals "$(completion_dir linux zsh)" "$DATA_HOME/zsh/site-functions" "linux zsh completion_dir"
assert_equals "$(completion_dir linux fish)" "$CONFIG_HOME/fish/completions" "linux fish completion_dir"
assert_equals "$(completion_dir darwin bash)" "/usr/local/etc/bash_completion.d" "darwin bash completion_dir"
assert_equals "$(completion_dir darwin zsh)" "/usr/local/share/zsh/site-functions" "darwin zsh completion_dir"
assert_equals "$(completion_dir darwin fish)" "$CONFIG_HOME/fish/completions" "darwin fish completion_dir"

CUSTOM_DATA_HOME="/tmp/container-use-test-xdg-data"
CUSTOM_CONFIG_HOME="/tmp/container-use-test-xdg-config"
mkdir -p "$CUSTOM_DATA_HOME" "$CUSTOM_CONFIG_HOME"
XDG_DATA_HOME_BACKUP="${XDG_DATA_HOME:-}"
XDG_CONFIG_HOME_BACKUP="${XDG_CONFIG_HOME:-}"
export XDG_DATA_HOME="$CUSTOM_DATA_HOME"
export XDG_CONFIG_HOME="$CUSTOM_CONFIG_HOME"

assert_equals "$(completion_path linux bash container-use)" "$CUSTOM_DATA_HOME/bash-completion/completions/container-use" "linux bash container-use with XDG_DATA_HOME"
assert_equals "$(completion_path linux zsh container-use)" "$CUSTOM_DATA_HOME/zsh/site-functions/_container-use" "linux zsh container-use with XDG_DATA_HOME"
assert_equals "$(completion_path linux fish container-use)" "$CUSTOM_CONFIG_HOME/fish/completions/container-use.fish" "linux fish container-use with XDG_CONFIG_HOME"
assert_equals "$(completion_dir linux bash)" "$CUSTOM_DATA_HOME/bash-completion/completions" "linux bash completion_dir with XDG_DATA_HOME"
assert_equals "$(completion_dir linux zsh)" "$CUSTOM_DATA_HOME/zsh/site-functions" "linux zsh completion_dir with XDG_DATA_HOME"
assert_equals "$(completion_dir linux fish)" "$CUSTOM_CONFIG_HOME/fish/completions" "linux fish completion_dir with XDG_CONFIG_HOME"

if [ -n "$XDG_DATA_HOME_BACKUP" ]; then
    export XDG_DATA_HOME="$XDG_DATA_HOME_BACKUP"
else
    unset XDG_DATA_HOME
fi
if [ -n "$XDG_CONFIG_HOME_BACKUP" ]; then
    export XDG_CONFIG_HOME="$XDG_CONFIG_HOME_BACKUP"
else
    unset XDG_CONFIG_HOME
fi

detect_os() {
    echo "linux"
}

linux_instructions="$(show_completion_instructions "container-use")"
assert_contains "$linux_instructions" "mkdir -p $DATA_HOME/bash-completion/completions" "linux bash mkdirp instruction"
assert_contains "$linux_instructions" "mkdir -p $DATA_HOME/zsh/site-functions" "linux zsh mkdirp instruction"
assert_contains "$linux_instructions" "mkdir -p $CONFIG_HOME/fish/completions" "linux fish mkdirp instruction"
assert_contains "$linux_instructions" "container-use completion bash > $DATA_HOME/bash-completion/completions/container-use" "linux container-use bash redirection"
assert_contains "$linux_instructions" "container-use completion --command-name=cu fish > $CONFIG_HOME/fish/completions/cu.fish" "linux cu fish redirection"
unset -f detect_os

detect_os() {
    echo "darwin"
}

darwin_instructions="$(show_completion_instructions "container-use")"
assert_contains "$darwin_instructions" "mkdir -p /usr/local/etc/bash_completion.d" "darwin bash mkdirp instruction"
assert_contains "$darwin_instructions" "mkdir -p /usr/local/share/zsh/site-functions" "darwin zsh mkdirp instruction"
assert_contains "$darwin_instructions" "mkdir -p $HOME/.config/fish/completions" "darwin fish mkdirp instruction"
assert_contains "$darwin_instructions" "container-use completion bash > /usr/local/etc/bash_completion.d/container-use" "darwin container-use bash redirection"
unset -f detect_os

echo "✅ install.sh completion paths behave as expected"
