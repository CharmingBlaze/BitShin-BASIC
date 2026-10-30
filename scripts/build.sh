#!/bin/sh
set -e
cd "$(dirname "$0")/.."
go build -o moon ./cmd/bs
go build -o moonsls ./cmd/bsls
echo "Built moon and moonsls"
