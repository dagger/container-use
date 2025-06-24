package agentconfig

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dagger/container-use/mcpserver"
)

var claudeAgent = &Agent{
	Name:      "claude",
	RulesFile: "CLAUDE.md",
	Detect: func(dir string) bool {
		// Check for .claude directory or CLAUDE.md file
		if _, err := os.Stat(filepath.Join(dir, ".claude")); err == nil {
			return true
		}
		if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err == nil {
			return true
		}
		return false
	},
	ConfigureMCP: func(cmd string) error {
		c := exec.Command("claude", "mcp", "add", "container-use", "--", cmd, "stdio")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
	AutoApproveMCP: func(dir string) error {
		tools := []string{}
		for _, t := range mcpserver.Tools() {
			tools = append(tools, fmt.Sprintf("mcp__container-use__%s", t.Definition.Name))
		}
		return updateConfig(
			filepath.Join(dir, ".claude", "settings.local.json"),
			map[string]any{
				"permissions": map[string]any{
					"allow": tools,
				},
			},
			"json",
		)
	},
}
