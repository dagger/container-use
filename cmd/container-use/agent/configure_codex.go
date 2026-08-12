package agent

import (
	"fmt"
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

func (a *ConfigureCodex) name() string {
	return a.Name
}

func (a *ConfigureCodex) description() string {
	return a.Description
}

func (a *ConfigureCodex) editMcpConfig() error {
	configPath, err := homedir.Expand(filepath.Join("~", ".codex", "config.toml"))
	if err != nil {
		return err
	}

	return writeMcpConfig(configPath, func(data []byte, cfg *map[string]any) error {
		return toml.Unmarshal(data, cfg)
	}, a.updateCodexConfig)
}

func (a *ConfigureCodex) updateCodexConfig(config map[string]any) ([]byte, error) {
	if config == nil {
		config = make(map[string]any)
	}

	var mcpServers map[string]any
	if servers, ok := config["mcp_servers"]; ok {
		mcpServers = servers.(map[string]any)
	} else {
		mcpServers = make(map[string]any)
		config["mcp_servers"] = mcpServers
	}

	mcpServers["container-use"] = map[string]any{
		"command":      ContainerUseBinary,
		"args":         []any{"stdio"},
		"auto_approve": tools(""),
	}

	data, err := toml.Marshal(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	return data, nil
}

func (a *ConfigureCodex) editRules() error {
	agentsFile := "AGENTS.md"
	return saveRulesFile(agentsFile, rules.AgentRules)
}

func (a *ConfigureCodex) isInstalled() bool {
	_, err := exec.LookPath("codex")
	return err == nil
}
