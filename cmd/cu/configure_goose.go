package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

// Configuration structures for different agents
type GooseExtension struct {
	Name    string            `yaml:"name"`
	Type    string            `yaml:"type"`
	Enabled bool              `yaml:"enabled"`
	Cmd     string            `yaml:"cmd"`
	Args    []string          `yaml:"args"`
	Envs    map[string]string `yaml:"envs"`
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
	configPath, err := homedir.Expand(filepath.Join("~", ".config", "goose", "config.yaml"))
	if err != nil {
		return err
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

	// Get extensions map
	var extensions map[string]any
	if ext, ok := config["extensions"]; ok {
		extensions = ext.(map[string]any)
	} else {
		extensions = make(map[string]any)
		config["extensions"] = extensions
	}

	cuPath, err := exec.LookPath(CU_BINARY)
	if err != nil {
		return fmt.Errorf("cu command not found in PATH: %w", err)
	}

	// Add container-use extension
	extensions["container-use"] = map[string]any{
		"name":    "container-use",
		"type":    "stdio",
		"enabled": true,
		"cmd":     cuPath,
		"args":    []any{"stdio"},
		"envs":    map[string]any{},
	}

	// Write config back
	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

// Save the agent rules with the container-use prompt
func (a *ConfigureGoose) editRules() error {
	gooseHints, err := homedir.Expand(filepath.Join("~", ".config", "goose", ".goosehints"))
	if err != nil {
		return err
	}
	return saveRulesFile(gooseHints, rules.AgentRules)
}

func (a *ConfigureGoose) isInstalled() bool {
	_, err := exec.LookPath("goose")
	if err != nil {
		return false
	}
	return true
}
