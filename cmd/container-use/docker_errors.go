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

	// Linux: Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?
	if strings.Contains(errStr, "cannot connect to the docker daemon") {
		return true
	}

	// Windows: error during connect: Get "http://%2F%2F.%2Fpipe%2FdockerDesktopLinuxEngine/v1.51/containers/json": open //./pipe/dockerDesktopLinuxEngine: The system cannot find the file specified.
	if strings.Contains(errStr, "error during connect") && strings.Contains(errStr, "pipe/dockerdesktoplinuxengine") && strings.Contains(errStr, "the system cannot find the file specified") {
		return true
	}

	// macOS: request returned 500 Internal Server Error for API route and version http://%2FUsers%2Fb1tank%2F.docker%2Frun%2Fdocker.sock/v1.50/containers/json, check if the server supports the requested API version
	if strings.Contains(errStr, "request returned 500 internal server error") && strings.Contains(errStr, "docker.sock") && strings.Contains(errStr, "check if the server supports the requested api version") {
		return true
	}

	// Generic fallbacks
	return strings.Contains(errStr, "docker daemon") ||
		strings.Contains(errStr, "docker.sock")
}

// handleDockerDaemonError prints a helpful error message for Docker daemon issues
func handleDockerDaemonError() {
	fmt.Fprintf(os.Stderr, "\nError: Docker daemon is not running.\n")
	fmt.Fprintf(os.Stderr, "Please start Docker and try again.\n\n")
}
