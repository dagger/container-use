package agent

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/dagger/container-use/rules"
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

func (a *ConfigureQ) name() string {
	return a.Name
}

func (a *ConfigureQ) description() string {
	return a.Description
}

func (a *ConfigureQ) editMcpConfig() error {
	return writeMcpConfig(filepath.Join(".amazonq", "mcp.json"), func(data []byte, cfg *MCPServersConfig) error {
		return json.Unmarshal(data, cfg)
	}, a.updateMcpConfig)
}

func (a *ConfigureQ) updateMcpConfig(config MCPServersConfig) ([]byte, error) {
	// Initialize mcpServers map if nil
	if config.MCPServers == nil {
		config.MCPServers = make(map[string]MCPServer)
	}

	config.MCPServers["container-use"] = MCPServer{
		Command: ContainerUseBinary,
		Args:    []string{"stdio"},
		Env:     map[string]string{},
		Timeout: &[]int{60000}[0],
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	return data, nil
}

func (a *ConfigureQ) editRules() error {
	return saveRulesFile(".amazonq/rules/container-use.md", rules.AgentRules)
}

func (a *ConfigureQ) isInstalled() bool {
	_, err := exec.LookPath("q")
	return err == nil
}
