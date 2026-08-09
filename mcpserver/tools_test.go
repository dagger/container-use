package mcpserver

import (
	"context"
	"encoding/json"
	"testing"

	"dagger.io/dagger"
	"github.com/dagger/container-use/environment"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentResponseFromEnvInfo(t *testing.T) {
	envInfo := &environment.EnvironmentInfo{
		ID: "adverb-animal",
		State: &environment.State{
			Title:  "my env",
			Config: environment.DefaultConfig(),
		},
	}

	resp := environmentResponseFromEnvInfo(envInfo)
	assert.Equal(t, "adverb-animal", resp.ID)
	assert.Equal(t, "my env", resp.Title)
	assert.Equal(t, "container-use/adverb-animal", resp.RemoteRef)
	assert.Equal(t, "container-use checkout adverb-animal", resp.CheckoutCommand)
	assert.Equal(t, "container-use log adverb-animal", resp.LogCommand)
	assert.Equal(t, "container-use diff adverb-animal", resp.DiffCommand)
}

func TestEnvironmentResponseFromEnv(t *testing.T) {
	env := &environment.Environment{
		EnvironmentInfo: &environment.EnvironmentInfo{
			ID:    "env-1",
			State: &environment.State{Title: "env one"},
		},
		Services: []*environment.Service{{Config: &environment.ServiceConfig{Name: "svc"}}},
	}

	resp := environmentResponseFromEnv(env)
	assert.Equal(t, "env-1", resp.ID)
	assert.Len(t, resp.Services, 1)
	assert.Equal(t, "svc", resp.Services[0].Config.Name)
}

func TestMarshalEnvironmentInfo(t *testing.T) {
	envInfo := &environment.EnvironmentInfo{
		ID:    "env-2",
		State: &environment.State{Title: "title two"},
	}

	out, err := marshalEnvironmentInfo(envInfo)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	assert.Equal(t, "env-2", decoded["id"])
	assert.Equal(t, "title two", decoded["title"])
}

func TestEnvironmentInfoToCallResult(t *testing.T) {
	envInfo := &environment.EnvironmentInfo{
		ID:    "env-3",
		State: &environment.State{Title: "title three"},
	}

	result, err := EnvironmentInfoToCallResult(envInfo)
	require.NoError(t, err)
	require.Len(t, result.Content, 1)
	text, ok := mcp.AsTextContent(result.Content[0])
	require.True(t, ok)
	assert.Contains(t, text.Text, "env-3")
}

func TestCreateTools(t *testing.T) {
	tools := createTools(false)
	assert.Len(t, tools, 15)

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Definition.Name] = true
	}
	assert.Contains(t, names, "environment_open")
	assert.Contains(t, names, "environment_create")
	assert.Contains(t, names, "environment_run_cmd")
	assert.Contains(t, names, "environment_log")
	assert.Contains(t, names, "environment_diff")
}

func TestWrapTool(t *testing.T) {
	called := false
	tool := createEnvironmentOpenTool()
	tool.Handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		called = true
		return mcp.NewToolResultText("ok"), nil
	}

	wrapped := wrapTool(tool)
	assert.Equal(t, tool.Definition.Name, wrapped.Definition.Name)

	_, err := wrapped.Handler(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestWrapToolWithClient(t *testing.T) {
	called := false
	sentinel := &dagger.Client{}
	tool := createEnvironmentOpenTool()
	tool.Handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dag, hasDag := ctx.Value(daggerClientKey{}).(*dagger.Client)
		_, hasSingleTenant := ctx.Value(singleTenantKey{}).(bool)
		called = true
		assert.True(t, hasDag)
		assert.Same(t, sentinel, dag)
		assert.True(t, hasSingleTenant)
		return mcp.NewToolResultText("ok"), nil
	}

	wrapped := wrapToolWithClient(tool, sentinel, true)
	_, err := wrapped.Handler(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	assert.True(t, called)
}
