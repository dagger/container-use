package main

import (
	"errors"
	"testing"
)

func TestIsDockerDaemonError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "docker daemon error - linux",
			err:      errors.New("Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?"),
			expected: true,
		},
		{
			name:     "docker daemon error - windows",
			err:      errors.New("error during connect: Get \"http://%2F%2F.%2Fpipe%2FdockerDesktopLinuxEngine/v1.51/containers/json\": open //./pipe/dockerDesktopLinuxEngine: The system cannot find the file specified."),
			expected: true,
		},
		{
			name:     "docker daemon error - macos",
			err:      errors.New("request returned 500 Internal Server Error for API route and version http://%2FUsers%2Fb1tank%2F.docker%2Frun%2Fdocker.sock/v1.50/containers/json, check if the server supports the requested API version"),
			expected: true,
		},
		{
			name:     "docker daemon error - generic",
			err:      errors.New("docker daemon is not running"),
			expected: true,
		},
		{
			name:     "docker socket error - generic",
			err:      errors.New("connection to docker.sock failed"),
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDockerDaemonError(tt.err); got != tt.expected {
				t.Errorf("isDockerDaemonError() = %v, want %v", got, tt.expected)
			}
		})
	}
}
