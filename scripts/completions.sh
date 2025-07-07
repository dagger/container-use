#!/bin/sh
set -e
rm -rf completions
mkdir completions

for sh in bash zsh fish; do
	go run ./cmd/container-use completion "$sh" >"completions/container-use.$sh"
	go run ./cmd/container-use completion --command-name=cu "$sh" >"completions/cu.$sh"
done
