package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCommand(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		checkOutput   func(t *testing.T, output string)
		expectedError bool
	}{
		{
			name: "basic version output",
			args: []string{"version"},
			checkOutput: func(t *testing.T, output string) {
				// Should always show version, may show commit and build date
				assert.Contains(t, output, "container-use version")
				// Should not show system info without --system
				assert.NotContains(t, output, "System:")
				assert.NotContains(t, output, "Container Runtime:")
				assert.NotContains(t, output, "Git:")
			},
		},
		{
			name: "system flag shows system info",
			args: []string{"version", "--system"},
			checkOutput: func(t *testing.T, output string) {
				// Should show basic version info
				assert.Contains(t, output, "container-use version")

				// Should show system info section
				assert.Contains(t, output, "System:")
				assert.Contains(t, output, "OS/Arch:")
				assert.Contains(t, output, "Container Runtime:")
				assert.Contains(t, output, "Git:")
				assert.Contains(t, output, "Dagger CLI:")

				// Should show OS/arch format
				assert.Regexp(t, `[\w]+/[\w]+`, output)

				// Container runtime output should show one of the supported runtimes
				// This handles: "Docker 24.0.5", "Podman 4.3.1", "Docker 24.0.5 (daemon not running)", or "not found"
				assert.Regexp(t, `Container Runtime: ((Docker|Podman|nerdctl|finch) [\d\.]+(v[\d\.]+)?(\s+\(daemon not running\))?|not found)`, output)
			},
		},
		{
			name: "short flag works",
			args: []string{"version", "-s"},
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "System:")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new command instance for each test
			cmd := rootCmd
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			output := buf.String()
			if tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestVersionParsing(t *testing.T) {
	// Test that version parsing handles common formats gracefully
	// This is a focused integration test of the parsing logic
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "docker standard format",
			input:    "Docker version 24.0.5, build 1234567",
			expected: "24.0.5",
		},
		{
			name:     "git standard format",
			input:    "git version 2.39.3",
			expected: "2.39.3",
		},
		{
			name:     "git with vendor info",
			input:    "git version 2.39.3 (Apple Git-145)",
			expected: "2.39.3",
		},
		{
			name:     "dagger with v prefix",
			input:    "dagger v0.21.7 (image://...)",
			expected: "0.21.7",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "unknown",
		},
		{
			name:     "unrelated output",
			input:    "command not found",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractVersion(tt.input))
		})
	}
}

func TestRuntimeInfoString(t *testing.T) {
	assert.Equal(t, "Docker 29.6.2", (&runtimeInfo{Name: "Docker", Version: "29.6.2", Running: true}).String())
	assert.Equal(t, "Podman unknown (daemon not running)", (&runtimeInfo{Name: "Podman", Version: "unknown", Running: false}).String())
}

func TestGetBuildInfoFromBinary(t *testing.T) {
	commit, date := getBuildInfoFromBinary()
	// In test builds this may be "unknown" or real values; just ensure no panic.
	assert.True(t, commit == "unknown" || len(commit) >= 7)
	assert.NotEmpty(t, date)
}
