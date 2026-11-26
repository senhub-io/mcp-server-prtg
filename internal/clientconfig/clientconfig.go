// Package clientconfig renders the settings MCP clients need to reach this server.
package clientconfig

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Host returns the host name MCP clients should use: override when set,
// otherwise the machine name.
func Host(override string) string {
	if override != "" {
		return override
	}
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		return hostname
	}
	return "localhost"
}

// Render writes ready-to-paste configurations for the supported MCP clients.
func Render(w io.Writer, host string, port string, tls bool, apiKey, certFile string) error {
	scheme := "http"
	if tls {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s:%s/mcp", scheme, host, port)

	writeJSON := func(v any) error {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}

	env := map[string]string{"MCP_PRTG_API_KEY": apiKey}
	if tls {
		env["NODE_EXTRA_CA_CERTS"] = "<path on the client computer to a copy of server.crt>"
	}

	if _, err := fmt.Fprintln(w, "Claude Desktop (claude_desktop_config.json):"); err != nil {
		return err
	}
	if err := writeJSON(map[string]any{
		"mcpServers": map[string]any{
			"prtg": map[string]any{
				"command": "npx",
				"args":    []string{"-y", "mcp-remote", url, "--header", "Authorization:Bearer ${MCP_PRTG_API_KEY}"},
				"env":     env,
			},
		},
	}); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "\nClaude Code:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "claude mcp add --transport http prtg %s --header \"Authorization: Bearer %s\"\n", url, apiKey); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "\nCursor (~/.cursor/mcp.json):"); err != nil {
		return err
	}
	if err := writeJSON(map[string]any{
		"mcpServers": map[string]any{
			"prtg": map[string]any{
				"url":     url,
				"headers": map[string]string{"Authorization": "Bearer " + apiKey},
			},
		},
	}); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "\nVS Code (.vscode/mcp.json):"); err != nil {
		return err
	}
	if err := writeJSON(map[string]any{
		"servers": map[string]any{
			"prtg": map[string]any{
				"type":    "http",
				"url":     url,
				"headers": map[string]string{"Authorization": "Bearer " + apiKey},
			},
		},
	}); err != nil {
		return err
	}

	if tls {
		if _, err := fmt.Fprintf(w, "\nTLS: copy the server certificate %s to each client computer. Claude Desktop trusts it through NODE_EXTRA_CA_CERTS above; for Claude Code, Cursor and VS Code, import it into the computer's trusted root certificates. Clients must use a host name or address the certificate covers (%s by default).\n", certFile, host); err != nil {
			return err
		}
	}

	return nil
}
