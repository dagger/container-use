#!/usr/bin/env bash
set -uoe pipefail

: {$TARGETPLATFORM:-local}
which docker >/dev/null || ( echo "Please follow instructions to install Docker at https://docs.docker.com/get-started/get-docker/"; exit 1 )
docker build --platform "$TARGETPLATFORM" -o . .
ls cu
