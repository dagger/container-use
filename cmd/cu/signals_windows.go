//go:build windows

package main

import (
	"os"
)

func setupPlatformSignals() {
	// Windows doesn't support SIGUSR1, so this is a no-op
}

func handleSIGUSR(sigusrCh <-chan os.Signal) {
	// Windows doesn't support SIGUSR1, so this is a no-op
}
