#!/bin/sh
set -e
cd "$(dirname "$0")/.."
go build -o bs ./cmd/bs
echo "Built bs"
