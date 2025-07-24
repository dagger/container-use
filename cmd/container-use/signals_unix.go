//go:build !windows

package main

import (
	"os"
	"syscall"
)

// getNotifySignals returns Unix-compatible signals
func getNotifySignals() []os.Signal {
	return []os.Signal{os.Interrupt, os.Kill, syscall.SIGTERM}
}
