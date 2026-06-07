#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$ROOT_DIR/bin"
CORE_BIN="$ROOT_DIR/core/target/debug/ap1_core"
API_BIN="$BIN_DIR/ap1-api"
CLI_BIN="$BIN_DIR/ap1-cli"

function ensure_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Errore: '$1' non trovato. Installa $1 prima di proseguire."
    exit 1
  fi
}

ensure_command cargo
ensure_command go

mkdir -p "$BIN_DIR"

echo "[1/4] Build core Rust..."
cd "$ROOT_DIR/core"
cargo build

echo "[2/4] Build API Go..."
cd "$ROOT_DIR/api"
go build -o "$API_BIN"

echo "[3/4] Build CLI Go..."
cd "$ROOT_DIR/cli"
go build -o "$CLI_BIN"

cd "$ROOT_DIR"

function cleanup() {
  echo "\nArresto AP1..."
  kill "${CORE_PID:-}" "${API_PID:-}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "[4/4] Avvio core e API in background..."
"$CORE_BIN" &
CORE_PID=$!
"$API_BIN" -config "$ROOT_DIR/config/global.yaml" -plugins "$ROOT_DIR/config/plugins.yaml" -addr ":8080" &
API_PID=$!

sleep 2

echo "AP1 avviato."
echo "  core PID = $CORE_PID"
echo "  api  PID = $API_PID"
echo "Usa '$CLI_BIN status' per verificare lo stato, oppure premi Ctrl+C per fermare tutto."

wait
