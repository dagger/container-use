package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/dagger/container-use/rules"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"
)

type ConfigureGoose struct {
	Name        string
	Description string
}

func NewConfigureGoose() *ConfigureGoose {
	return &ConfigureGoose{
		Name:        "Goose",
		Description: "an open source, extensible AI agent that goes beyond code suggestions",
	}
}

// Return the agents full name
func (a *ConfigureGoose) name() string {
	return a.Name
}

// Return a description of the agent
func (a *ConfigureGoose) description() string {
	return a.Description
}

// Save the MCP config with container-use enabled
func (a *ConfigureGoose) editMcpConfig() error {
	var configPath string
	var err error

	if runtime.GOOS == "windows" {
		// Windows: %APPDATA%\Block\goose\config\config.yaml
		// Reference: https://block.github.io/goose/docs/guides/config-file
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return fmt.Errorf("APPDATA environment variable not set")
		}
		configPath = filepath.Join(appData, "Block", "goose", "config", "config.yaml")
	} else {
		// macOS/Linux: ~/.config/goose/config.yaml
		configPath, err = homedir.Expand(filepath.Join("~", ".config", "goose", "config.yaml"))
		if err != nil {
			return err
		}
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Read existing config or create new
	var config map[string]any
	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]any)
	}

	data, err := a.updateGooseConfig(config)
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

func (a *ConfigureGoose) updateGooseConfig(config map[string]any) ([]byte, error) {
	// Get extensions map
	var extensions map[string]any
	if ext, ok := config["extensions"]; ok {
		extensions = ext.(map[string]any)
	} else {
		extensions = make(map[string]any)
		config["extensions"] = extensions
	}

	// Add container-use extension
	extensions["container-use"] = map[string]any{
		"name":    "container-use",
		"type":    "stdio",
		"enabled": true,
		"cmd":     ContainerUseBinary,
		"args":    []any{"stdio"},
		"envs":    map[string]any{},
	}

	// Write config back
	data, err := yaml.Marshal(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	return data, nil
}

// Save the agent rules with the container-use prompt
func (a *ConfigureGoose) editRules() error {
	return saveRulesFile(".goosehints", rules.AgentRules)
}

func (a *ConfigureGoose) isInstalled() bool {
	_, err := exec.LookPath("goose")
	return err == nil
}
