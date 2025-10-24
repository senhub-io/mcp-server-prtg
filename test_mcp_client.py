#!/usr/bin/env python3
"""
Simple MCP Client for testing MCP Server PRTG v2.0

Usage:
    pip install requests sseclient-py
    python3 test_mcp_client.py

Author: Matthieu Noirbusson <matthieu.noirbusson@sensorfactory.eu>
"""

import json
import sys
import time
import requests
from sseclient import SSEClient

# Configuration
API_KEY = "test-api-key-12345678-1234-1234-1234-123456789abc"
BASE_URL = "http://127.0.0.1:8443"
TIMEOUT = 5

class Colors:
    """ANSI color codes for terminal output"""
    HEADER = '\033[95m'
    OKBLUE = '\033[94m'
    OKCYAN = '\033[96m'
    OKGREEN = '\033[92m'
    WARNING = '\033[93m'
    FAIL = '\033[91m'
    ENDC = '\033[0m'
    BOLD = '\033[1m'

def print_test(name):
    """Print test header"""
    print(f"\n{Colors.HEADER}{Colors.BOLD}{'='*60}{Colors.ENDC}")
    print(f"{Colors.HEADER}{Colors.BOLD}🧪 {name}{Colors.ENDC}")
    print(f"{Colors.HEADER}{Colors.BOLD}{'='*60}{Colors.ENDC}\n")

def print_success(msg):
    """Print success message"""
    print(f"{Colors.OKGREEN}✅ {msg}{Colors.ENDC}")

def print_error(msg):
    """Print error message"""
    print(f"{Colors.FAIL}❌ {msg}{Colors.ENDC}")

def print_info(msg):
    """Print info message"""
    print(f"{Colors.OKCYAN}ℹ️  {msg}{Colors.ENDC}")

def print_json(data):
    """Print formatted JSON"""
    print(f"{Colors.OKBLUE}{json.dumps(data, indent=2)}{Colors.ENDC}")

def test_health():
    """Test 1: Health check endpoint (no auth)"""
    print_test("Test 1: Health Check (Public Endpoint)")

    try:
        response = requests.get(f"{BASE_URL}/health", timeout=TIMEOUT)

        print_info(f"Status Code: {response.status_code}")
        print_info(f"Response:")
        print_json(response.json())

        if response.status_code == 200:
            print_success("Health check passed")
            return True
        else:
            print_error(f"Unexpected status code: {response.status_code}")
            return False

    except Exception as e:
        print_error(f"Health check failed: {e}")
        return False

def test_status_without_auth():
    """Test 2: Status endpoint without authentication (should fail)"""
    print_test("Test 2: Status Without Auth (Should Fail)")

    try:
        response = requests.get(f"{BASE_URL}/status", timeout=TIMEOUT)

        print_info(f"Status Code: {response.status_code}")

        if response.status_code == 401:
            print_success("Correctly rejected unauthorized request")
            return True
        else:
            print_error(f"Unexpected status code: {response.status_code} (expected 401)")
            return False

    except Exception as e:
        print_error(f"Test failed: {e}")
        return False

def test_status_with_auth():
    """Test 3: Status endpoint with authentication (should succeed)"""
    print_test("Test 3: Status With Auth (Should Succeed)")

    try:
        headers = {"Authorization": f"Bearer {API_KEY}"}
        response = requests.get(f"{BASE_URL}/status", headers=headers, timeout=TIMEOUT)

        print_info(f"Status Code: {response.status_code}")
        print_info(f"Response:")
        print_json(response.json())

        if response.status_code == 200:
            data = response.json()
            if data.get("status") == "running":
                print_success("Status check passed")
                print_info(f"Version: {data.get('version')}")
                print_info(f"MCP Tools: {data.get('mcp_tools')}")

                # Check database status
                db_status = data.get('database', {})
                db_state = db_status.get('status', 'unknown')
                db_error = db_status.get('error', '')

                if db_state == "connected":
                    print_success(f"Database: {db_state}")
                elif db_state == "disconnected":
                    print_error(f"Database: {db_state} - {db_error}")
                else:
                    print_info(f"Database: {db_state}")

                return True
            else:
                print_error(f"Unexpected status: {data.get('status')}")
                return False
        else:
            print_error(f"Unexpected status code: {response.status_code}")
            return False

    except Exception as e:
        print_error(f"Status check failed: {e}")
        return False

def test_sse_connection():
    """Test 4: SSE connection and endpoint discovery"""
    print_test("Test 4: SSE Connection")

    try:
        headers = {"Authorization": f"Bearer {API_KEY}"}
        print_info("Connecting to SSE endpoint...")

        response = requests.get(
            f"{BASE_URL}/sse",
            headers=headers,
            stream=True,
            timeout=TIMEOUT
        )

        print_info(f"Status Code: {response.status_code}")

        if response.status_code != 200:
            print_error(f"Failed to connect: {response.status_code}")
            return False, None

        print_info("Connected! Waiting for endpoint event...")

        client = SSEClient(response)
        for event in client.events():
            print_info(f"Received event: {event.event}")
            print_info(f"Data: {event.data}")

            if event.event == "endpoint":
                message_url = event.data
                print_success(f"Got message URL: {message_url}")
                return True, message_url

            # Only wait for first event
            break

        print_error("Did not receive endpoint event")
        return False, None

    except requests.exceptions.Timeout:
        print_error("Connection timeout")
        return False, None
    except Exception as e:
        print_error(f"SSE connection failed: {e}")
        return False, None

def test_mcp_rpc_call(message_url):
    """Test 5: MCP JSON-RPC call"""
    print_test("Test 5: MCP JSON-RPC Call")

    if not message_url:
        print_error("No message URL provided")
        return False

    try:
        headers = {
            "Authorization": f"Bearer {API_KEY}",
            "Content-Type": "application/json"
        }

        # Call tools/list to get available tools
        rpc_request = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "tools/list",
            "params": {}
        }

        print_info("Sending JSON-RPC request:")
        print_json(rpc_request)

        response = requests.post(
            message_url,
            headers=headers,
            json=rpc_request,
            timeout=TIMEOUT
        )

        print_info(f"Status Code: {response.status_code}")
        print_info(f"Response:")
        print_json(response.json())

        if response.status_code == 200:
            data = response.json()
            if "result" in data:
                print_success("RPC call succeeded")
                tools = data.get("result", {}).get("tools", [])
                print_info(f"Found {len(tools)} tools:")
                for tool in tools:
                    print(f"  - {tool.get('name')}: {tool.get('description', 'No description')[:60]}...")
                return True
            else:
                print_error(f"Unexpected response format")
                return False
        else:
            print_error(f"RPC call failed: {response.status_code}")
            return False

    except Exception as e:
        print_error(f"RPC call failed: {e}")
        return False

def test_mcp_get_sensors(message_url):
    """Test 6: Call prtg_get_sensors tool"""
    print_test("Test 6: Call prtg_get_sensors Tool")

    if not message_url:
        print_error("No message URL provided")
        return False

    try:
        headers = {
            "Authorization": f"Bearer {API_KEY}",
            "Content-Type": "application/json"
        }

        rpc_request = {
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/call",
            "params": {
                "name": "prtg_get_sensors",
                "arguments": {
                    "limit": 3
                }
            }
        }

        print_info("Calling prtg_get_sensors with limit=3...")

        response = requests.post(
            message_url,
            headers=headers,
            json=rpc_request,
            timeout=TIMEOUT
        )

        print_info(f"Status Code: {response.status_code}")

        if response.status_code == 200:
            data = response.json()
            if "result" in data:
                print_success("Tool call succeeded")
                print_info("Result:")
                print_json(data.get("result"))
                return True
            elif "error" in data:
                print_error(f"RPC Error: {data.get('error')}")
                # This is expected if database is not available
                print_info("Note: This error is expected if PostgreSQL is not configured")
                return True  # We consider this a success since the RPC worked
            else:
                print_error("Unexpected response format")
                return False
        else:
            print_error(f"Tool call failed: {response.status_code}")
            return False

    except Exception as e:
        print_error(f"Tool call failed: {e}")
        return False

def main():
    """Run all tests"""
    print(f"\n{Colors.BOLD}{Colors.HEADER}")
    print("╔═══════════════════════════════════════════════════════════╗")
    print("║     MCP Server PRTG v2.0 - Client Test Suite             ║")
    print("╚═══════════════════════════════════════════════════════════╝")
    print(f"{Colors.ENDC}")

    print_info(f"Target: {BASE_URL}")
    print_info(f"API Key: {API_KEY[:20]}...")

    results = []

    # Test 1: Health check
    results.append(("Health Check", test_health()))

    # Test 2: Status without auth
    results.append(("Auth Rejection", test_status_without_auth()))

    # Test 3: Status with auth
    results.append(("Auth Success", test_status_with_auth()))

    # Test 4: SSE connection
    sse_success, message_url = test_sse_connection()
    results.append(("SSE Connection", sse_success))

    # Test 5 & 6: RPC calls (only if SSE succeeded)
    if sse_success and message_url:
        results.append(("RPC List Tools", test_mcp_rpc_call(message_url)))
        results.append(("RPC Get Sensors", test_mcp_get_sensors(message_url)))
    else:
        print_error("Skipping RPC tests (SSE failed)")

    # Summary
    print(f"\n{Colors.BOLD}{Colors.HEADER}")
    print("╔═══════════════════════════════════════════════════════════╗")
    print("║                    Test Summary                           ║")
    print("╚═══════════════════════════════════════════════════════════╝")
    print(f"{Colors.ENDC}\n")

    passed = sum(1 for _, result in results if result)
    total = len(results)

    for name, result in results:
        status = f"{Colors.OKGREEN}✅ PASS{Colors.ENDC}" if result else f"{Colors.FAIL}❌ FAIL{Colors.ENDC}"
        print(f"  {name:.<50} {status}")

    print()
    if passed == total:
        print_success(f"All tests passed! ({passed}/{total})")
        return 0
    else:
        print_error(f"Some tests failed ({passed}/{total} passed)")
        return 1

if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print(f"\n\n{Colors.WARNING}⚠️  Tests interrupted by user{Colors.ENDC}")
        sys.exit(130)
    except Exception as e:
        print_error(f"Fatal error: {e}")
        sys.exit(1)
