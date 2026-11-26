package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/senhub-io/mcp-server-prtg/internal/cliargs"
)

func renderToString(t *testing.T, host, port string, tls bool, apiKey string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := renderClientConfig(&buf, host, port, tls, apiKey); err != nil {
		t.Fatalf("renderClientConfig: %v", err)
	}
	return buf.String()
}

func TestRenderClientConfigTLS(t *testing.T) {
	out := renderToString(t, "prtg.example.com", "8443", true, "secret-key")

	for _, want := range []string{
		"Claude Desktop (claude_desktop_config.json):",
		"Claude Code:",
		"Cursor (~/.cursor/mcp.json):",
		"VS Code (.vscode/mcp.json):",
		"https://prtg.example.com:8443/mcp",
		"NODE_TLS_REJECT_UNAUTHORIZED",
		"self-signed certificate",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}

	if strings.Count(out, "secret-key") != 4 {
		t.Errorf("expected API key in 4 places, got %d", strings.Count(out, "secret-key"))
	}
	if strings.Contains(out, "http://prtg.example.com") {
		t.Error("unexpected http URL in TLS output")
	}
}

func TestRenderClientConfigNoTLS(t *testing.T) {
	out := renderToString(t, "prtg.example.com", "8080", false, "secret-key")

	if !strings.Contains(out, "http://prtg.example.com:8080/mcp") {
		t.Error("missing http URL")
	}
	if strings.Contains(out, "https://") {
		t.Error("unexpected https URL in non-TLS output")
	}
	if strings.Contains(out, "NODE_TLS_REJECT_UNAUTHORIZED") {
		t.Error("NODE_TLS_REJECT_UNAUTHORIZED must not appear when tls is false")
	}
	if strings.Contains(out, "self-signed certificate") {
		t.Error("TLS note must not appear when tls is false")
	}
}

func TestRenderClientConfigKeyVerbatim(t *testing.T) {
	key := "a&<>b"
	out := renderToString(t, "host", "8443", true, key)
	if strings.Count(out, key) != 4 {
		t.Errorf("key with special chars not printed verbatim, count=%d", strings.Count(out, key))
	}
	if strings.Contains(out, "\\u0026") || strings.Contains(out, "\\u003c") {
		t.Error("HTML escaping mangled the key")
	}
}

func TestRenderClientConfigDesktopJSONParses(t *testing.T) {
	out := renderToString(t, "host", "8443", true, "k")
	start := strings.Index(out, "Claude Desktop (claude_desktop_config.json):")
	if start < 0 {
		t.Fatal("missing Claude Desktop section")
	}
	rest := out[start:]
	end := strings.Index(rest, "\n\nClaude Code:")
	if end < 0 {
		t.Fatal("missing Claude Code section")
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(rest[len("Claude Desktop (claude_desktop_config.json):")+1:end]), &parsed); err != nil {
		t.Fatalf("Claude Desktop JSON does not parse: %v", err)
	}
	servers, ok := parsed["mcpServers"].(map[string]any)
	if !ok || servers["prtg"] == nil {
		t.Error("parsed JSON missing mcpServers.prtg")
	}
}

func TestDefaultConfigPathExplicit(t *testing.T) {
	args := &cliargs.ParsedArgs{ConfigPath: "/tmp/custom-config.yaml"}
	if got := defaultConfigPath(args); got != "/tmp/custom-config.yaml" {
		t.Errorf("expected explicit path unchanged, got %q", got)
	}
}
