package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dagger/container-use/mcpserver"
	"github.com/spf13/cobra"
)

type MCPServersConfig struct {
	MCPServers map[string]MCPServer `json:"mcpServers"`
}

type MCPServer struct {
	Command       string            `json:"command"`
	Args          []string          `json:"args"`
	Env           map[string]string `json:"env,omitempty"`
	Timeout       *int              `json:"timeout,omitempty"`
	Disabled      *bool             `json:"disabled,omitempty"`
	AutoApprove   []string          `json:"autoApprove,omitempty"`
	AlwaysAllow   []string          `json:"alwaysAllow,omitempty"`
	WorkingDir    *string           `json:"working_directory,omitempty"`
	StartOnLaunch *bool             `json:"start_on_launch,omitempty"`
}

const ContainerUseBinary = "container-use"

var configureCmd = &cobra.Command{
	Use:   "agent [agent]",
	Short: "Configure MCP server for different agents",
	Long:  `Setup the container-use MCP server according to the specified agent including Claude Code, Goose, Cursor, and others.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return interactiveConfiguration()
		}
		agent, err := selectAgent(args[0])
		if err != nil {
			return err
		}
		return configureAgent(agent)
	},
}

func interactiveConfiguration() error {
	selectedAgent, err := RunAgentSelector()
	if err != nil {
		// If the user quits, it's not an error, just exit gracefully.
		if err.Error() == "no agent selected" {
			return nil
		}
		return fmt.Errorf("failed to select agent: %w", err)
	}

	agent, err := selectAgent(selectedAgent)
	if err != nil {
		return err
	}
	return configureAgent(agent)
}

type ConfigurableAgent interface {
	name() string
	description() string
	editMcpConfig() error
	editRules() error
	isInstalled() bool
}

// Add agents here
func selectAgent(agentKey string) (ConfigurableAgent, error) {
	switch agentKey {
	case "claude":
		return &ConfigureClaude{}, nil
	case "goose":
		return &ConfigureGoose{}, nil
	case "cursor":
		return &ConfigureCursor{}, nil
	case "codex":
		return &ConfigureCodex{}, nil
	case "amazonq":
		return &ConfigureQ{}, nil
	}
	return nil, fmt.Errorf("unknown agent: %s", agentKey)
}

func configureAgent(agent ConfigurableAgent) error {
	fmt.Printf("Configuring %s...\n", agent.name())

	// Save MCP config
	err := agent.editMcpConfig()
	if err != nil {
		return err
	}
	fmt.Printf("✓ Configured %s MCP configuration\n", agent.name())

	// Save rules
	err = agent.editRules()
	if err != nil {
		return err
	}
	fmt.Printf("✓ Saved %s container-use rules\n", agent.name())

	fmt.Printf("\n%s configuration complete!\n", agent.name())
	return nil
}

// Helper functions
func saveRulesFile(rulesFile, content string) error {
	dir := filepath.Dir(rulesFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Append to file if it exists, create if it doesn't TODO make it re-entrant with a marker
	existing, err := os.ReadFile(rulesFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read existing rules: %w", err)
	}
	existingStr := string(existing)

	editedRules, err := editRulesFile(existingStr, content)
	if err != nil {
		return err
	}

	err = os.WriteFile(rulesFile, []byte(editedRules), 0644)
	if err != nil {
		return fmt.Errorf("failed to update rules: %w", err)
	}

	return nil
}

func editRulesFile(existingRules, content string) (string, error) {
	// Look for section markers
	const marker = "<!-- container-use-rules -->"

	if strings.Contains(existingRules, marker) {
		// Update existing section
		parts := strings.Split(existingRules, marker)
		if len(parts) != 3 {
			return "", fmt.Errorf("malformed rules file - expected single section marked with %s", marker)
		}
		newContent := parts[0] + marker + "\n" + content + "\n" + marker + parts[2]
		return newContent, nil
	} else {
		// Append new section
		newContent := existingRules
		if len(newContent) > 0 && !strings.HasSuffix(newContent, "\n") {
			newContent += "\n"
		}
		newContent += "\n" + marker + "\n" + content + "\n" + marker + "\n"
		return newContent, nil
	}
}

func tools(prefix string) []string {
	tools := []string{}
	for _, t := range mcpserver.Tools() {
		tools = append(tools, fmt.Sprintf("%s%s", prefix, t.Definition.Name))
	}
	return tools
}

func init() {
	configCmd.AddCommand(configureCmd)
}
