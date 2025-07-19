//go:build !windows

package mcpserver

import (
	"os"
	"syscall"
)

// getNotifySignals returns Unix-compatible signals for MCP server
func getNotifySignals() []os.Signal {
	return []os.Signal{os.Interrupt, os.Kill, syscall.SIGTERM}
}
