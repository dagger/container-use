package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dagger/container-use/rules"
	"github.com/mitchellh/go-homedir"
)

type ConfigureQ struct {
	Name        string
	Description string
}

func NewConfigureQ() *ConfigureQ {
	return &ConfigureQ{
		Name:        "Amazon Q Developer",
		Description: "Amazon's agentic chat experience in your terminal",
	}
}

// Return the agents full name
func (a *ConfigureQ) name() string {
	return a.Name
}

// Return a description of the agent
func (a *ConfigureQ) description() string {
	return a.Description
}

// Save the MCP config with container-use enabled
func (a *ConfigureQ) editMcpConfig() error {
	configPath, err := homedir.Expand(filepath.Join("~", ".aws", "amazonq", "mcp.json"))
	if err != nil {
		return err
	}

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
		Env:     map[string]string{},
		Timeout: &[]int{60000}[0], // TODO: configure trusted tools
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
func (a *ConfigureQ) editRules() error {
	return saveRulesFile(".amazonq/rules/container-use.md", rules.AgentRules)
}

func (a *ConfigureQ) isInstalled() bool {
	_, err := exec.LookPath("q")
	return err == nil
}
