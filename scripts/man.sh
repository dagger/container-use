#!/bin/sh
set -e
rm -rf man
mkdir man
go run ./cmd/container-use man > "man/container-use.1"
