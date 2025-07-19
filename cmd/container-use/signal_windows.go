//go:build windows

package main

// On Windows, SIGUSR1 is not available, so we provide a no-op implementation
func setupSignalHandling() {
	// No special signal handling on Windows
}
