#!/usr/bin/env python3

"""
MCP Server PRTG - SSE Test Script (using requests)
Usage: python3 test-mcp-sse-v2.py
"""

import requests
import json
import re
import sys
import threading
import time

# Disable SSL warnings for self-signed certificates
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

API_KEY = "3a4e8cb4-1066-4b18-85f5-4f46ecefc4d1"
BASE_URL = "https://dash999.hibouvision.com:8443"

session_id = None
sse_responses = []
sse_connected = False

def print_header():
    print("╔════════════════════════════════════════════════════════════════╗")
    print("║         MCP Server PRTG - SSE Connection Test                  ║")
    print("╚════════════════════════════════════════════════════════════════╝")
    print()

def sse_listener():
    """Keep SSE connection open and listen for responses"""
    global session_id, sse_responses, sse_connected

    try:
        print(f"   📡 Opening connection to {BASE_URL}/sse...")

        headers = {
            "Authorization": f"Bearer {API_KEY}",
            "Accept": "text/event-stream",
            "Cache-Control": "no-cache"
        }

        # Use requests with streaming
        response = requests.get(
            f"{BASE_URL}/sse",
            headers=headers,
            stream=True,
            verify=False,
            timeout=30
        )

        print(f"   ✅ Connected! Status: {response.status_code}")
        print(f"   Headers: {dict(response.headers)}")
        print()

        sse_connected = True

        # Read SSE events line by line
        line_count = 0
        for line in response.iter_lines(decode_unicode=True):
            if line is None:
                continue

            line = line.strip()
            line_count += 1

            if line:
                print(f"   [Line {line_count}] {line}")

            # Extract sessionId from first event
            if session_id is None and "sessionId=" in line:
                match = re.search(r'sessionId=([a-f0-9\-]+)', line)
                if match:
                    session_id = match.group(1)
                    print(f"   ✅ Found sessionId: {session_id}")
                    print()

            # Collect all SSE messages (JSON-RPC responses)
            if line.startswith("data:"):
                data = line[5:].strip()
                if data and not data.startswith("http"):
                    # This is a JSON-RPC response
                    sse_responses.append(data)
                    print(f"   📩 SSE Response received")
                    print()

    except requests.exceptions.Timeout:
        print(f"   ⏱️  SSE connection timed out")
    except Exception as e:
        print(f"   ❌ SSE Error: {type(e).__name__}: {e}")
        import traceback
        traceback.print_exc()

def test_mcp_flow():
    """Test the complete MCP flow"""
    global session_id, sse_responses, sse_connected

    print("🔌 Step 1: Opening SSE connection...")
    print(f"   URL: {BASE_URL}/sse")
    print()

    # Start SSE listener in background
    sse_thread = threading.Thread(target=sse_listener, daemon=True)
    sse_thread.start()

    # Wait for connection
    print("   Waiting for SSE connection...")
    for i in range(10):
        if sse_connected:
            break
        time.sleep(0.5)

    if not sse_connected:
        print("❌ Failed to connect to SSE endpoint")
        return False

    # Wait for sessionId to be extracted
    print("   Waiting for sessionId...")
    for i in range(10):
        if session_id:
            break
        time.sleep(0.5)

    if not session_id:
        print("❌ Failed to get sessionId from SSE connection")
        time.sleep(1)
        return False

    print(f"✅ Session ID: {session_id}")
    print()

    # Step 2: Send RPC request
    print("🔧 Step 2: Sending tools/list request...")
    print(f"   URL: {BASE_URL}/message?sessionId={session_id}")
    print()

    try:
        payload = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "tools/list"
        }

        headers = {
            "Authorization": f"Bearer {API_KEY}",
            "Content-Type": "application/json"
        }

        response = requests.post(
            f"{BASE_URL}/message",
            params={"sessionId": session_id},
            json=payload,
            headers=headers,
            verify=False,
            timeout=10
        )

        print(f"   POST Response Status: {response.status_code}")

        if response.status_code in [200, 202]:
            print("   ✅ Request accepted, waiting for SSE response...")
            print()

            # Wait for response via SSE
            print("📨 Step 3: Listening for response on SSE connection...")
            for i in range(10):
                if sse_responses:
                    break
                time.sleep(0.5)

            if sse_responses:
                result = json.loads(sse_responses[0])
                print()
                print("Response:")
                print(json.dumps(result, indent=2))
                print()

                if "result" in result and "tools" in result["result"]:
                    tools = result["result"]["tools"]
                    print(f"✅ Success! Found {len(tools)} MCP tools")
                    print()
                    print("Available tools:")
                    for tool in tools:
                        print(f"  - {tool['name']}: {tool['description']}")
                    return True
                else:
                    print("❌ Unexpected response format")
                    return False
            else:
                print("❌ No response received via SSE")
                return False
        else:
            print(f"❌ Unexpected status: {response.status_code}")
            print(f"   Response: {response.text}")
            return False

    except Exception as e:
        print(f"❌ Error: {type(e).__name__}: {e}")
        return False

def main():
    print_header()

    success = test_mcp_flow()

    print()
    print("╔════════════════════════════════════════════════════════════════╗")
    if success:
        print("║                   ✅ Test Complete - SUCCESS                   ║")
    else:
        print("║                   ❌ Test Complete - FAILED                    ║")
    print("╚════════════════════════════════════════════════════════════════╝")

    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()
