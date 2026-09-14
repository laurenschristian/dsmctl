#!/bin/sh
# Emit shell completion scripts into completions/ (used by goreleaser + make docs).
set -e
mkdir -p completions
go run . completion bash > completions/dsmctl.bash
go run . completion zsh  > completions/_dsmctl
go run . completion fish > completions/dsmctl.fish
