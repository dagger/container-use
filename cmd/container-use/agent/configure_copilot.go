package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dagger/container-use/rules"
	"github.com/mitchellh/go-homedir"
)

type ConfigureCopilot struct {
	Name        string
	Description string
}

func NewConfigureCopilot() *ConfigureCopilot {
	return &ConfigureCopilot{
		Name:        "GitHub Copilot CLI",
		Description: "GitHub's AI-powered coding assistant in your terminal",
	}
}

// Return the agents full name
func (a *ConfigureCopilot) name() string {
	return a.Name
}

// Return a description of the agent
func (a *ConfigureCopilot) description() string {
	return a.Description
}

// Save the MCP config with container-use enabled
func (a *ConfigureCopilot) editMcpConfig() error {
	configPath, err := homedir.Expand(filepath.Join("~", ".copilot", "mcp-config.json"))
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
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]any)
	}

	data, err := a.updateMcpConfig(config)
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

func (a *ConfigureCopilot) updateMcpConfig(config map[string]any) ([]byte, error) {
	// Get mcpServers map, preserving any other servers the user already has
	var mcpServers map[string]any
	if servers, ok := config["mcpServers"].(map[string]any); ok {
		mcpServers = servers
	} else {
		mcpServers = make(map[string]any)
		config["mcpServers"] = mcpServers
	}

	// Add container-use server
	mcpServers["container-use"] = map[string]any{
		"type":    "local",
		"command": ContainerUseBinary,
		"args":    []any{"stdio"},
		"tools":   []any{"*"},
	}

	// Write config back
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	return data, nil
}

// Save the agent rules with the container-use prompt
func (a *ConfigureCopilot) editRules() error {
	rulesFile := filepath.Join(".github", "copilot-instructions.md")
	return saveRulesFile(rulesFile, rules.AgentRules)
}

func (a *ConfigureCopilot) isInstalled() bool {
	_, err := exec.LookPath("copilot")
	return err == nil
}
