package environment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnvironmentConfig_Load verifies the best-effort loading behavior
// The Load method should gracefully handle missing files while still failing on actual errors
func TestEnvironmentConfig_Load(t *testing.T) {
	scenarios := []struct {
		name            string
		setup           func(t *testing.T, dir string)
		expectError     bool
		expectBaseImage string
		expectWorkdir   string
	}{
		{
			name:            "both_files_missing",
			setup:           func(t *testing.T, dir string) {}, // no setup
			expectError:     false,
			expectBaseImage: "ubuntu:24.04",
			expectWorkdir:   "/workdir",
		},
		{
			name: "only_instructions_missing",
			setup: func(t *testing.T, dir string) {
				createConfigFile(t, dir, &EnvironmentConfig{
					BaseImage: "custom:image",
					Workdir:   "/custom",
				})
			},
			expectError:     false,
			expectBaseImage: "custom:image",
			expectWorkdir:   "/custom",
		},
		{
			name: "only_environment_missing",
			setup: func(t *testing.T, dir string) {
				createInstructionsFile(t, dir, "Custom instructions")
			},
			expectError:     false,
			expectBaseImage: "ubuntu:24.04",
			expectWorkdir:   "/workdir",
		},
		{
			name: "both_files_present",
			setup: func(t *testing.T, dir string) {
				createInstructionsFile(t, dir, "Test instructions")
				createConfigFile(t, dir, &EnvironmentConfig{
					BaseImage: "test:image",
					Workdir:   "/test",
				})
			},
			expectError:     false,
			expectBaseImage: "test:image",
			expectWorkdir:   "/test",
		},
		{
			name: "invalid_json",
			setup: func(t *testing.T, dir string) {
				configDir := filepath.Join(dir, ".container-use")
				require.NoError(t, os.MkdirAll(configDir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(configDir, "environment.json"), []byte("invalid json"), 0644))
			},
			expectError: true,
		},
		{
			name: "config_directory_permission_error",
			setup: func(t *testing.T, dir string) {
				if os.Getuid() == 0 {
					t.Skip("Skipping permission test as root")
				}
				if runtime.GOOS == "windows" {
					t.Skip("Skipping permission test on Windows - Windows file permissions work differently")
				}
				configDir := filepath.Join(dir, ".container-use")
				require.NoError(t, os.MkdirAll(configDir, 0000))
				t.Cleanup(func() { os.Chmod(configDir, 0755) })
			},
			expectError: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tempDir := t.TempDir()
			config := DefaultConfig()

			scenario.setup(t, tempDir)

			err := config.Load(tempDir)

			if scenario.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, scenario.expectBaseImage, config.BaseImage)
			assert.Equal(t, scenario.expectWorkdir, config.Workdir)
		})
	}
}

// TestEnvironmentConfig_PreservesShellOperators tests that shell operators like && are not
// escaped as unicode sequences when saving and loading configuration
func TestEnvironmentConfig_PreservesShellOperators(t *testing.T) {
	tempDir := t.TempDir()

	// Create a config with shell operators in setup commands
	originalConfig := &EnvironmentConfig{
		BaseImage: "ubuntu:24.04",
		Workdir:   "/workdir",
		SetupCommands: []string{
			"apt update && apt install -y python3",
			"echo 'test' && ls -la",
			"cd /tmp && wget https://example.com/file.tar.gz && tar -xzf file.tar.gz",
		},
	}

	// Save the configuration
	err := originalConfig.Save(tempDir)
	require.NoError(t, err)

	// Read the raw JSON file to check for unicode escaping
	configPath := filepath.Join(tempDir, ".container-use", "environment.json")
	rawData, err := os.ReadFile(configPath)
	require.NoError(t, err)

	rawJSON := string(rawData)

	// Check that && is NOT escaped as \u0026\u0026
	assert.NotContains(t, rawJSON, "\\u0026\\u0026", "Shell operators should not be unicode-escaped in saved JSON")
	assert.Contains(t, rawJSON, "&&", "Shell operators should be preserved as-is")

	// Load the configuration back
	loadedConfig := DefaultConfig()
	err = loadedConfig.Load(tempDir)
	require.NoError(t, err)

	// Verify that the loaded configuration preserves the shell operators
	require.Equal(t, len(originalConfig.SetupCommands), len(loadedConfig.SetupCommands))
	for i, cmd := range originalConfig.SetupCommands {
		assert.Equal(t, cmd, loadedConfig.SetupCommands[i], "Setup command %d should preserve shell operators", i)
		assert.Contains(t, loadedConfig.SetupCommands[i], "&&", "Setup command should contain original && operator")
		assert.NotContains(t, loadedConfig.SetupCommands[i], "\\u0026", "Setup command should not contain unicode-escaped &")
	}
}

// Test helper functions
func createInstructionsFile(t *testing.T, dir, content string) {
	t.Helper()
	configDir := filepath.Join(dir, ".container-use")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "AGENT.md"), []byte(content), 0644))
}

func createConfigFile(t *testing.T, dir string, config *EnvironmentConfig) {
	t.Helper()
	configDir := filepath.Join(dir, ".container-use")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	data, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "environment.json"), data, 0644))
}
