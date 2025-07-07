package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dagger/container-use/rules"
	"github.com/mitchellh/go-homedir"
	"github.com/pelletier/go-toml/v2"
)

type ConfigureCodex struct {
	Name        string
	Description string
}

func NewConfigureCodex() *ConfigureCodex {
	return &ConfigureCodex{
		Name:        "OpenAI Codex",
		Description: "OpenAI's lightweight coding agent that runs in your terminal",
	}
}

// TOML structures for Codex configuration
type CodexConfig struct {
	MCPServers map[string]CodexMCPServer `toml:"mcp_servers"`
}

type CodexMCPServer struct {
	Command string            `toml:"command"`
	Args    []string          `toml:"args"`
	Env     map[string]string `toml:"env"`
}

// Return the agents full name
func (a *ConfigureCodex) name() string {
	return a.Name
}

// Return a description of the agent
func (a *ConfigureCodex) description() string {
	return a.Description
}

// Save the MCP config with container-use enabled
func (a *ConfigureCodex) editMcpConfig() error {
	configPath, err := homedir.Expand(filepath.Join("~", ".codex", "config.toml"))
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
		if err := toml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]any)
	}

	// Get mcp_servers map
	var mcpServers map[string]any
	if servers, ok := config["mcp_servers"]; ok {
		mcpServers = servers.(map[string]any)
	} else {
		mcpServers = make(map[string]any)
		config["mcp_servers"] = mcpServers
	}

	cuPath, err := exec.LookPath(CU_BINARY)
	if err != nil {
		return fmt.Errorf("cu command not found in PATH: %w", err)
	}

	// Add container-use server
	mcpServers["container-use"] = map[string]any{
		"command":      cuPath,
		"args":         []any{"stdio"},
		"auto_approve": tools(""),
	}

	// Write config back
	data, err := toml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

// Save the agent rules with the container-use prompt
func (a *ConfigureCodex) editRules() error {
	agentsFile := "AGENTS.md"
	return saveRulesFile(agentsFile, rules.AgentRules)
}

func (a *ConfigureCodex) isInstalled() bool {
	_, err := exec.LookPath("codex")
	if err != nil {
		return false
	}
	return true
}
