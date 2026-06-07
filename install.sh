#!/usr/bin/env bash
set -euo pipefail

echo "AP1 install bootstrap"

if ! command -v cargo >/dev/null 2>&1; then
  echo "cargo non trovato. Installa Rust prima di proseguire."
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go non trovato. Installa Go prima di proseguire."
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

echo "AP1 bootstrap completo. Usa './ap1' per avviare core e API con un solo comando, oppure usa 'make core' e 'make api' per avviarne i singoli componenti."
