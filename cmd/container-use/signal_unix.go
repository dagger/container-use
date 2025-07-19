//go:build !windows

package main

import (
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func setupSignalHandling() {
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

func dumpStacks() {
	buf := make([]byte, 1<<20) // 1MB buffer
	n := runtime.Stack(buf, true)
	io.MultiWriter(logWriter, os.Stderr).Write(buf[:n])
}
