//go:build windows

package mcpserver

import (
	"os"
	"syscall"
)

// getNotifySignals returns Windows-compatible signals for MCP server
func getNotifySignals() []os.Signal {
	// On Windows:
	// - os.Interrupt: Ctrl+C signal (can receive, cannot send to other processes)
	// - syscall.SIGTERM: Termination signal (can receive)
	// - os.Kill: Not a receivable signal, used only for Process.Kill()
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
