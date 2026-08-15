#!/bin/sh
set -e
cd "$(dirname "$0")/.."
go test ./internal/...
go vet ./internal/lex ./internal/parse ./internal/interp ./internal/value ./internal/ast ./internal/phys2d ./internal/phys3d ./internal/netenet
