package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/senhub-io/mcp-server-prtg/internal/cliargs"
)

func defaultConfigPath(args *cliargs.ParsedArgs) string {
	if args.ConfigPath != "./config.yaml" {
		return args.ConfigPath
	}
	if _, err := os.Stat(args.ConfigPath); err == nil {
		return args.ConfigPath
	}
	var installer string
	if runtime.GOOS == "windows" {
		installer = filepath.Join(os.Getenv("ProgramData"), "Sensor Factory", "MCP Server PRTG", "config.yaml")
	} else {
		installer = "/etc/mcp-server-prtg/config.yaml"
	}
	if _, err := os.Stat(installer); err == nil {
		return installer
	}
	return args.ConfigPath
}

func clientHost(override string) string {
	if override != "" {
		return override
	}
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		return hostname
	}
	return "localhost"
}

func renderClientConfig(w io.Writer, host string, port string, tls bool, apiKey string) error {
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
		env["NODE_TLS_REJECT_UNAUTHORIZED"] = "0"
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
		if _, err := fmt.Fprintln(w, "\nThe server uses a self-signed certificate by default: Claude Desktop accepts it through NODE_TLS_REJECT_UNAUTHORIZED=0 above; for the other clients install a certificate they trust or add the server certificate to the client machine's trust store."); err != nil {
			return err
		}
	}

	return nil
}
