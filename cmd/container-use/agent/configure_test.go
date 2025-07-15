package agent

import (
	"testing"

	"github.com/dagger/container-use/rules"
	"github.com/stretchr/testify/assert"
)

func TestConfigureEditRulesFile(t *testing.T) {
	blankRules := "\nFOO\n"
	existingRules := blankRules + rules.AgentRules

	// Edit rules file that doesnt have container-use rules yet
	editedBlank, err := editRulesFile(blankRules, rules.AgentRules)
	assert.NoError(t, err)
	// check rules have been added
	assert.Contains(t, editedBlank, rules.AgentRules)
	// check original content is still there
	assert.Contains(t, editedBlank, blankRules)

	// Edit rules file that has existing container-use rules
	editedExisting, err := editRulesFile(existingRules, rules.AgentRules)
	assert.NoError(t, err)
	assert.Contains(t, editedExisting, rules.AgentRules)
}

func TestConfigureClaudeUpdateSettings(t *testing.T) {
	claude := &ConfigureClaude{}
	settings := ClaudeSettingsLocal{}
	expect := `{
  "permissions": {
    "allow": [
      "mcp__container-use__environment_`
	editedSettings, err := claude.updateSettingsLocal(settings)
	assert.NoError(t, err)
	assert.Contains(t, string(editedSettings), expect)
}

func TestConfigureCodexUpdateConfig(t *testing.T) {
	codex := &ConfigureCodex{}
	config := make(map[string]any)
	contains := `[mcp_servers]
[mcp_servers.container-use]
args = ['stdio']
auto_approve = ['`
	editedConfig, err := codex.updateCodexConfig(config)
	assert.NoError(t, err)
	assert.Contains(t, string(editedConfig), contains)
}

func TestConfigureGooseUpdateConfig(t *testing.T) {
	goose := &ConfigureGoose{}
	config := make(map[string]any)
	contains := `extensions:
    container-use:
        args:
            - stdio
        cmd: container-use
        enabled: true
        envs: {}
        name: container-use
        type: stdio`
	editedConfig, err := goose.updateGooseConfig(config)
	assert.NoError(t, err)
	assert.Contains(t, string(editedConfig), contains)
}

func TestConfigureQUpdateConfig(t *testing.T) {
	q := &ConfigureQ{}
	config := MCPServersConfig{}
	expect := `{
  "mcpServers": {
    "container-use": {
      "command": "container-use",
      "args": [
        "stdio"
      ],
      "timeout": 60000
    }
  }
}`
	editedConfig, err := q.updateMcpConfig(config)
	assert.NoError(t, err)
	assert.Equal(t, string(editedConfig), expect)
}

func TestConfigureCursorUpdateConfig(t *testing.T) {
	q := &ConfigureCursor{}
	config := MCPServersConfig{}
	expect := `{
  "mcpServers": {
    "container-use": {
      "command": "container-use",
      "args": [
        "stdio"
      ]
    }
  }
}`
	editedConfig, err := q.updateMcpConfig(config)
	assert.NoError(t, err)
	assert.Equal(t, string(editedConfig), expect)
}
