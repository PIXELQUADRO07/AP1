#!/usr/bin/env bash
# AP1 Optimized Startup Script
set -eo pipefail

# Colors
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$ROOT_DIR/bin"
LOG_DIR="$ROOT_DIR/system/runtime/logs"
CORE_BIN="$ROOT_DIR/core/target/debug/ap1_core"
API_BIN="$BIN_DIR/ap1-api"
CLI_BIN="$ROOT_DIR/ap1-cli" # Linked to root for ease of use

REBUILD=false
if [[ "${1:-}" == "--rebuild" ]]; then
    REBUILD=true
fi

mkdir -p "$BIN_DIR" "$LOG_DIR"

echo -e "${CYAN}"
cat << "EOF"
 .+"+.+"+.+"+.+"+.+"+.+"+.+"+.+"+.+"+.+"+.
(        _           ____         _       )
 )      / \         |  _ \       / |     (
(      / _ \        | |_) |      | |      )
 )    / ___ \       |  __/       | |     (
(    /_/   \_\      |_|          |_|      )
 )                                       (
(                                         )
 "+.+"+.+"+.+"+.+"+.+"+.+"+.+"+.+"+.+"+.+"
EOF
echo -e "      AP1 - Edge-Aware Orchestrator${NC}\n"

# 1. Build Phase (Always Rebuild/Check)
echo -n -e "${YELLOW}[>] Building/Updating components... ${NC}"

# Core
cd "$ROOT_DIR/core"
cargo build -q > /dev/null 2>&1 || (echo -e "${RED}Core Build Failed${NC}" && exit 1)

# API
cd "$ROOT_DIR/api"
go build -o "$API_BIN" main.go > /dev/null 2>&1 || (echo -e "${RED}API Build Failed${NC}" && exit 1)

# CLI
cd "$ROOT_DIR/cli"
go build -o "$CLI_BIN" main.go repl.go tui.go > /dev/null 2>&1 || (echo -e "${RED}CLI Build Failed${NC}" && exit 1)

echo -e "${GREEN}Done!${NC}"

# 2. Cleanup function
function cleanup() {
  echo -e "\n${RED}[!] Stopping AP1 services...${NC}"
  kill "${CORE_PID:-}" "${API_PID:-}" 2>/dev/null || true
  exit 0
}
trap cleanup EXIT INT TERM

# 3. Execution Phase
echo -n -e "${YELLOW}[>] Starting Core & API in background... ${NC}"

# Start Core
cd "$ROOT_DIR/core"
"$CORE_BIN" > "$LOG_DIR/core.log" 2>&1 &
CORE_PID=$!

# Start API
cd "$ROOT_DIR/api"
"$API_BIN" -config "$ROOT_DIR/config/global.yaml" -plugins "$ROOT_DIR/config/plugins.yaml" -addr ":8080" > "$LOG_DIR/api.log" 2>&1 &
API_PID=$!

# Give it a second to initialize
sleep 1.5

if ps -p $CORE_PID > /dev/null && ps -p $API_PID > /dev/null; then
    echo -e "${GREEN}Running!${NC}"
    echo -e "${CYAN}[*] Core Log: system/runtime/logs/core.log${NC}"
    echo -e "${CYAN}[*] API Log:  system/runtime/logs/api.log${NC}"
    echo -e "--------------------------------------------------------"
    echo -e "${GREEN}AP1 is ready. Launching interactive CLI...${NC}"
    echo -e "--------------------------------------------------------"

    # 4. Launch CLI automatically
    sudo "$CLI_BIN" interactive
else
    echo -e "${RED}Failed!${NC}"
    echo -e "Check logs in $LOG_DIR for details."
    exit 1
fi
