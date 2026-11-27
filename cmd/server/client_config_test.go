package main

import (
	"testing"

	"github.com/senhub-io/mcp-server-prtg/internal/cliargs"
)

func TestDefaultConfigPathExplicit(t *testing.T) {
	args := &cliargs.ParsedArgs{ConfigPath: "/tmp/custom-config.yaml"}
	if got := defaultConfigPath(args); got != "/tmp/custom-config.yaml" {
		t.Errorf("expected explicit path unchanged, got %q", got)
	}
}
