//go:build unix

package main

import (
	"os"
	"os/signal"
	"syscall"
)

func setupPlatformSignals() {
	sigusrCh := make(chan os.Signal, 1)
	signal.Notify(sigusrCh, syscall.SIGUSR1)
	go handleSIGUSR(sigusrCh)
}

func handleSIGUSR(sigusrCh <-chan os.Signal) {
	for sig := range sigusrCh {
		if sig == syscall.SIGUSR1 {
			dumpStacks()
		}
	}
}
