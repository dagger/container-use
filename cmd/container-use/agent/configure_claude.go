package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dagger/container-use/rules"
)

type ConfigureClaude struct {
	Name        string
	Description string
}

func NewConfigureClaude() *ConfigureClaude {
	return &ConfigureClaude{
		Name:        "Claude Code",
		Description: "Anthropic's Claude Code",
	}
}

type ClaudeSettingsLocal struct {
	Permissions *ClaudePermissions `json:"permissions,omitempty"`
	Env         map[string]string  `json:"env,omitempty"`
}

type ClaudePermissions struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

func (c *ConfigureClaude) name() string {
	return c.Name
}

func (c *ConfigureClaude) description() string {
	return c.Description
}

func (c *ConfigureClaude) editMcpConfig() error {
	// Add MCP server
	cmd := exec.Command("claude", "mcp", "add", "container-use", "--", ContainerUseBinary, "stdio")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("could not automatically add MCP server: %w", err)
	}

	// Configure auto approve settings
	configPath := filepath.Join(".claude", "settings.local.json")
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	var config ClaudeSettingsLocal
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	}

	data, err := c.updateSettingsLocal(config)
	if err != nil {
		return err
	}

	err = os.WriteFile(configPath, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

func (c *ConfigureClaude) updateSettingsLocal(config ClaudeSettingsLocal) ([]byte, error) {
	// Initialize permissions map if nil
	if config.Permissions == nil {
		config.Permissions = &ClaudePermissions{Allow: []string{}}
	}

	// remove save non-container-use items from allow
	allows := []string{}
	for _, tool := range config.Permissions.Allow {
		if !strings.HasPrefix(tool, "mcp__container-use") {
			allows = append(allows, tool)
		}
	}

	// Add container-use tools to allow
	tools := tools("mcp__container-use__")
	allows = append(allows, tools...)
	config.Permissions.Allow = allows

	// Write config back
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	return data, nil
}

func (c *ConfigureClaude) editRules() error {
	return saveRulesFile("CLAUDE.md", rules.AgentRules)
}

func (c *ConfigureClaude) isInstalled() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}
