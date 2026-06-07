#!/usr/bin/env bash
set -euo pipefail

echo "AP1 install bootstrap"

if ! command -v cargo >/dev/null 2>&1; then
  echo "cargo not found. Install Rust before continuing."
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go not found. Install Go before continuing."
  exit 1
fi

cd "$(dirname "$0")"

echo "+ Install API dependencies"
cd api
go mod tidy
cd ..

echo "+ Check Rust core"
cd core
cargo check
cd ..

echo "+ Build CLI"
cd cli
go build -o ../ap1-cli
cd ..

echo "AP1 bootstrap complete. Use './ap1' to start core and API with one command, or use 'make core' and 'make api' to start components separately."
