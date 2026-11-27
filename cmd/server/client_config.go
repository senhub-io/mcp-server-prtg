package main

import (
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
