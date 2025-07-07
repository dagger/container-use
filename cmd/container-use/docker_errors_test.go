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
			name:     "docker daemon error",
			err:      errors.New("Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?"),
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
