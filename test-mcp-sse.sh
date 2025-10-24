#!/bin/bash

# MCP Server PRTG - SSE Test Script
# Usage: ./test-mcp-sse.sh

set -e

API_KEY="3a4e8cb4-1066-4b18-85f5-4f46ecefc4d1"
BASE_URL="https://dash999.hibouvision.com:8443"

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║         MCP Server PRTG - SSE Connection Test                  ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Step 1: Connect to SSE endpoint and extract sessionId
echo "🔌 Step 1: Connecting to SSE endpoint..."
echo "   URL: $BASE_URL/sse"
echo ""

# Use timeout to prevent hanging, and extract sessionId
# --no-buffer forces curl to output immediately
SSE_RESPONSE=$(timeout 3 curl -k -N --no-buffer -H "Authorization: Bearer $API_KEY" "$BASE_URL/sse" 2>&1 | head -n 10)

echo "Raw SSE Response:"
echo "$SSE_RESPONSE"
echo ""

SESSION_ID=$(echo "$SSE_RESPONSE" | grep "sessionId=" | sed 's/.*sessionId=\([^&[:space:]]*\).*/\1/' | head -1)

if [ -z "$SESSION_ID" ]; then
    echo "❌ Failed to extract sessionId from SSE response"
    exit 1
fi

echo "✅ Session ID extracted: $SESSION_ID"
echo ""

# Step 2: Test tools/list
echo "🔧 Step 2: Calling tools/list..."
echo "   URL: $BASE_URL/message?sessionId=$SESSION_ID"
echo ""

TOOLS_RESPONSE=$(curl -k -X POST "$BASE_URL/message?sessionId=$SESSION_ID" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list"
  }' 2>/dev/null)

echo "Response:"
echo "$TOOLS_RESPONSE" | jq .
echo ""

# Check if we got tools
TOOL_COUNT=$(echo "$TOOLS_RESPONSE" | jq -r '.result.tools // [] | length')

if [ "$TOOL_COUNT" -gt 0 ]; then
    echo "✅ Success! Found $TOOL_COUNT MCP tools"
    echo ""
    echo "Available tools:"
    echo "$TOOLS_RESPONSE" | jq -r '.result.tools[] | "  - \(.name): \(.description)"'
else
    echo "❌ No tools found or error occurred"
fi

echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                        Test Complete                           ║"
echo "╚════════════════════════════════════════════════════════════════╝"
