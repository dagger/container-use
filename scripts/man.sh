#!/bin/sh
set -e
rm -rf man
mkdir man
go run ./cmd/cu man > "man/container-use.1"
