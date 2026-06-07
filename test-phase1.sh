#!/bin/bash

# AP1 Phase 1 Test Script
# Tests basic API functionality (requires sudo for full network test)

set -e

API_URL="http://localhost:8080"
API_PORT=8080
WLAN_IFACE="wlan0"

echo "=== AP1 Phase 1 Functionality Test ==="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to test endpoint
test_endpoint() {
    local method=$1
    local path=$2
    local data=$3
    local expected_code=$4

    echo -n "Testing $method $path ... "

    if [ -z "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$API_URL$path")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" -H "Content-Type: application/json" -d "$data" "$API_URL$path")
    fi

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)

    if [ "$http_code" = "$expected_code" ]; then
        echo -e "${GREEN}✓ OK${NC} (HTTP $http_code)"
        echo "  Response: $(echo "$body" | head -c 100)..."
    else
        echo -e "${RED}✗ FAILED${NC} (Expected $expected_code, got $http_code)"
        echo "  Response: $body"
        return 1
    fi
    echo ""
}

echo "Step 1: Start AP1 API server..."
cd /home/gaetal/github/AP1/api
./ap1-api -config ../config/global.yaml -plugins ../config/plugins.yaml -addr :8080 &
API_PID=$!

sleep 2
echo "API Server started with PID $API_PID"
echo ""

# Trap to kill server on exit
trap "kill $API_PID 2>/dev/null" EXIT

echo "Step 2: Test basic health endpoints..."
test_endpoint "GET" "/health" "" "200"
test_endpoint "GET" "/api/config" "" "200"
test_endpoint "GET" "/api/profiles" "" "200"
echo ""

echo "Step 3: Test profile selection..."
test_endpoint "POST" "/api/profiles/select" '{"profile":"default"}' "200"
echo ""

echo "Step 4: Test firewall endpoints..."
test_endpoint "POST" "/api/system/firewall/apply" '{"interface":"wlan0","portal_ip":"192.168.50.1"}' "200"
test_endpoint "POST" "/api/system/firewall/clear" '{"interface":"wlan0"}' "200"
echo ""

echo "Step 5: Test interface configuration..."
test_endpoint "POST" "/api/system/interface/configure" '{"interface":"wlan0","ip":"192.168.50.1","subnet":"24"}' "200"
echo ""

echo "Step 6: Test portal status..."
test_endpoint "GET" "/api/portal/status" "" "200"
echo ""

echo -e "${GREEN}=== Phase 1 Tests Complete ===${NC}"
echo ""
echo "Integration Test Results:"
echo "- ✓ API server compiles and runs"
echo "- ✓ Health checks work"
echo "- ✓ Config endpoints accessible"
echo "- ✓ Profile selection endpoint works"
echo "- ✓ Firewall rule endpoints respond"
echo "- ✓ Interface configuration endpoint works"
echo "- ✓ Portal status endpoint works"
echo ""
echo "Note: Full network functionality (iptables, interface assignment) requires:"
echo "  - Running as root/sudo"
echo "  - Valid network interface (wlan0 or configured interface)"
echo "  - hostapd and dnsmasq installed"
echo ""
echo "Next steps:"
echo "  1. Run as root to test network functionality"
echo "  2. Start hostapd with generated config"
echo "  3. Connect to AP and test captive portal redirect"
