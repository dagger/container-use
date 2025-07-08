package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dagger/container-use/rules"
)

type ConfigureCursor struct {
	Name        string
	Description string
}

func NewConfigureCursor() *ConfigureCursor {
	return &ConfigureCursor{
		Name:        "Cursor",
		Description: "AI-powered code editor",
	}
}

// Return the agents full name
func (a *ConfigureCursor) name() string {
	return a.Name
}

// Return a description of the agent
func (a *ConfigureCursor) description() string {
	return a.Description
}

// Save the MCP config with container-use enabled
func (a *ConfigureCursor) editMcpConfig() error {
	configPath := filepath.Join(".cursor", "mcp.json")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Read existing config or create new
	var config MCPServersConfig
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	}

	// Initialize mcpServers map if nil
	if config.MCPServers == nil {
		config.MCPServers = make(map[string]MCPServer)
	}

	cuPath, err := exec.LookPath(ContainerUseBinary)
	if err != nil {
		return fmt.Errorf("cu command not found in PATH: %w", err)
	}

	// Add container-use server
	config.MCPServers["container-use"] = MCPServer{
		Command: cuPath,
		Args:    []string{"stdio"},
	}

	// Write config back
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

// Save the agent rules with the container-use prompt
func (a *ConfigureCursor) editRules() error {
	rulesFile := filepath.Join(".cursor", "rules", "container-use.mdc")
	return saveRulesFile(rulesFile, rules.CursorRules)
}

func (a *ConfigureCursor) isInstalled() bool {
	return true
}
