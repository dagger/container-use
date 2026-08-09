package agent

import (
	"encoding/json"
	"fmt"
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
	removeCmd := exec.Command("claude", "mcp", "remove", "container-use")
	_ = removeCmd.Run()

	cmd := exec.Command("claude", "mcp", "add", "container-use", "--", ContainerUseBinary, "stdio")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not automatically add MCP server: %w", err)
	}

	configPath := filepath.Join(".claude", "settings.local.json")
	return writeMcpConfig(configPath, func(data []byte, cfg *ClaudeSettingsLocal) error {
		return json.Unmarshal(data, cfg)
	}, c.updateSettingsLocal)
}

func (c *ConfigureClaude) updateSettingsLocal(config ClaudeSettingsLocal) ([]byte, error) {
	if config.Permissions == nil {
		config.Permissions = &ClaudePermissions{Allow: []string{}}
	}

	allows := []string{}
	for _, tool := range config.Permissions.Allow {
		if !strings.HasPrefix(tool, "mcp__container-use") {
			allows = append(allows, tool)
		}
	}

	tools := tools("mcp__container-use__")
	allows = append(allows, tools...)
	config.Permissions.Allow = allows

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
