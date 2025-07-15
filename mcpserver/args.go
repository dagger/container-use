package mcpserver

import (
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

const (
	repositoryToolSuffix  = "You MUST tell the user how to view environment changes using \"container-use log <environment_id>\", \"container-use diff <environment_id>\", AND \"container-use checkout <env_id>\". Failure to do so will make your work completely inaccessible."
	environmentToolSuffix = "You must call `environment_create` or `environment_open` to obtain a valid environment_id value. LLM-generated environment IDs WILL cause task failure."
)

var (
	explanationArgument = mcp.WithString("explanation",
		mcp.Description("One sentence explanation for why this directory is being listed."),
	)
	environmentSourceArgument = mcp.WithString("environment_source",
		mcp.Description("Absolute path to the source git repository for the environment."),
		mcp.Required(),
	)
	environmentIDArgument = mcp.WithString("environment_id",
		mcp.Description("The ID of the environment for this command. DO NOT generate environment_id values."),
		mcp.Required(),
	)
)

func newRepositoryTool(name string, description string, args ...mcp.ToolOption) mcp.Tool {
	opts := []mcp.ToolOption{
		mcp.WithDescription(strings.Join([]string{description, repositoryToolSuffix}, "\n\n")),
		explanationArgument,
		environmentSourceArgument,
	}
	opts = append(opts, args...)

	return mcp.NewTool(name, opts...)
}

func newEnvironmentTool(name string, description string, args ...mcp.ToolOption) mcp.Tool {
	opts := []mcp.ToolOption{
		mcp.WithDescription(strings.Join([]string{description, environmentToolSuffix}, "\n\n")),
		explanationArgument,
		environmentSourceArgument,
		environmentIDArgument,
	}
	opts = append(opts, args...)

	return mcp.NewTool(name, opts...)
}
