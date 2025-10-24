#!/usr/bin/env python3

"""
MCP Server PRTG - SSE Test Script
Usage: python3 test-mcp-sse.py
"""

import urllib3
import json
import re
import sys
import threading
import time
from urllib.request import Request, urlopen
from urllib.error import HTTPError, URLError

# Disable SSL warnings for self-signed certificates
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

API_KEY = "3a4e8cb4-1066-4b18-85f5-4f46ecefc4d1"
BASE_URL = "https://dash999.hibouvision.com:8443"

session_id = None
sse_responses = []

def print_header():
    print("╔════════════════════════════════════════════════════════════════╗")
    print("║         MCP Server PRTG - SSE Connection Test                  ║")
    print("╚════════════════════════════════════════════════════════════════╝")
    print()

def sse_listener():
    """Keep SSE connection open and listen for responses"""
    global session_id, sse_responses

    try:
        import ssl
        context = ssl._create_unverified_context()

        req = Request(f"{BASE_URL}/sse")
        req.add_header("Authorization", f"Bearer {API_KEY}")
        req.add_header("Accept", "text/event-stream")
        req.add_header("Cache-Control", "no-cache")

        print(f"   📡 Opening connection to {BASE_URL}/sse...")
        response = urlopen(req, context=context, timeout=30)
        print(f"   ✅ Connected! Status: {response.status}")
        print(f"   Headers: {dict(response.headers)}")
        print()

        # Read SSE events
        line_count = 0
        while True:
            line = response.readline().decode('utf-8').strip()
            line_count += 1

            print(f"   [Line {line_count}] Raw: '{line}'")

            if not line:
                continue

            # Extract sessionId from first event
            if session_id is None and "sessionId=" in line:
                match = re.search(r'sessionId=([a-f0-9\-]+)', line)
                if match:
                    session_id = match.group(1)
                    print(f"   ✅ Found sessionId: {session_id}")

            # Collect all SSE messages
            if line.startswith("data:"):
                data = line[5:].strip()
                if data and not data.startswith("http"):
                    # This is a JSON-RPC response
                    sse_responses.append(data)
                    print(f"   📩 SSE Response: {data[:100]}...")

            # Stop after reading enough lines if we have sessionId
            if session_id and line_count > 20:
                break

    except Exception as e:
        print(f"   ❌ SSE Error: {type(e).__name__}: {e}")
        import traceback
        traceback.print_exc()

def test_mcp_flow():
    """Test the complete MCP flow"""
    global session_id, sse_responses

    print("🔌 Step 1: Opening SSE connection...")
    print(f"   URL: {BASE_URL}/sse")
    print()

    # Start SSE listener in background
    sse_thread = threading.Thread(target=sse_listener, daemon=True)
    sse_thread.start()

    # Wait for sessionId to be extracted
    print("   Waiting for sessionId...")
    for i in range(20):  # Wait up to 10 seconds
        print(f"   ... {i}/20 (thread alive: {sse_thread.is_alive()})")
        if session_id:
            break
        time.sleep(0.5)

    if not session_id:
        print("❌ Failed to get sessionId from SSE connection")
        print(f"   Thread still alive: {sse_thread.is_alive()}")
        # Give thread a bit more time to print error messages
        time.sleep(2)
        return False

    print(f"✅ Session ID: {session_id}")
    print()

    # Step 2: Send RPC request
    print("🔧 Step 2: Sending tools/list request...")
    print(f"   URL: {BASE_URL}/message?sessionId={session_id}")
    print()

    try:
        import ssl
        context = ssl._create_unverified_context()

        payload = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "tools/list"
        }

        req = Request(
            f"{BASE_URL}/message?sessionId={session_id}",
            data=json.dumps(payload).encode('utf-8'),
            headers={
                "Authorization": f"Bearer {API_KEY}",
                "Content-Type": "application/json"
            },
            method="POST"
        )

        with urlopen(req, context=context, timeout=10) as response:
            status = response.status
            print(f"   POST Response Status: {status}")

            if status == 202 or status == 200:
                print("   ✅ Request accepted, waiting for SSE response...")
                print()

                # Wait for response via SSE
                print("📨 Step 3: Listening for response on SSE connection...")
                time.sleep(2)

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
                print(f"❌ Unexpected status: {status}")
                return False

    except HTTPError as e:
        print(f"❌ HTTP Error: {e.code} - {e.reason}")
        return False
    except Exception as e:
        print(f"❌ Error: {e}")
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
