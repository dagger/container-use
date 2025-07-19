//go:build windows

package main

import (
	"os"
	"syscall"
)

// getNotifySignals returns Windows-compatible signals for receiving
func getNotifySignals() []os.Signal {
	// On Windows:
	// - os.Interrupt: Ctrl+C signal (can receive, cannot send to other processes)
	// - syscall.SIGTERM: Termination signal (can receive) (https://pkg.go.dev/os/signal#hdr-Windows)
	// - os.Kill: Not a receivable signal
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
