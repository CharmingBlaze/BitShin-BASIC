#!/bin/sh
set -e
cd "$(dirname "$0")/.."
go build -o moon ./cmd/moon
go build -o moonsls ./cmd/moonsls
echo "Built moon and moonsls"
