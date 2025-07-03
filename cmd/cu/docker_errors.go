package main

import (
	"fmt"
	"os"
	"strings"
)

// isDockerDaemonError checks if the error is related to Docker daemon connectivity
func isDockerDaemonError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "cannot connect to the docker daemon") ||
		strings.Contains(errStr, "docker daemon") ||
		strings.Contains(errStr, "docker.sock")
}

// handleDockerDaemonError prints a helpful error message for Docker daemon issues
func handleDockerDaemonError() {
	fmt.Fprintf(os.Stderr, "\nError: Docker daemon is not running.\n")
	fmt.Fprintf(os.Stderr, "Please start Docker and try again.\n\n")
}
