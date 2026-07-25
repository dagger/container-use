package mcpserver

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func TestNewRepositoryTool(t *testing.T) {
	tool := newRepositoryTool("repo_test", "description")
	assert.Equal(t, "repo_test", tool.Name)
	assert.Contains(t, tool.Description, "description")

	schema := tool.InputSchema
	assert.NotNil(t, schema)
	assert.Contains(t, schema.Properties, "explanation")
	assert.Contains(t, schema.Properties, "environment_source")
}

func TestNewEnvironmentToolMultiTenant(t *testing.T) {
	tool := newEnvironmentTool(envToolOptions{
		name:        "env_test",
		description: "desc",
	}, mcp.WithString("command", mcp.Required()))

	assert.Equal(t, "env_test", tool.Name)
	schema := tool.InputSchema
	assert.Contains(t, schema.Properties, "explanation")
	assert.Contains(t, schema.Properties, "environment_source")
	assert.Contains(t, schema.Properties, "environment_id")
	assert.Contains(t, schema.Properties, "command")
}

func TestNewEnvironmentToolSingleTenant(t *testing.T) {
	tool := newEnvironmentTool(envToolOptions{
		name:                  "env_single",
		description:           "desc",
		useCurrentEnvironment: true,
	})

	schema := tool.InputSchema
	assert.Contains(t, schema.Properties, "explanation")
	assert.NotContains(t, schema.Properties, "environment_source")
	assert.NotContains(t, schema.Properties, "environment_id")
}
