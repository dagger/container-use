package mcpserver

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

var (
	explainationArgument = mcp.WithString("explanation",
		mcp.Description("One sentence explanation for why this directory is being listed."),
	)
	environmentSourceArgument = mcp.WithString("environment_source",
		mcp.Description("Absolute path to the source git repository for the environment."),
		mcp.Required(),
	)
	environmentIDArgument = mcp.WithString("environment_id",
		mcp.Description("The ID of the environment for this command."),
		mcp.Required(),
	)
)

func newRepositoryTool(name string, description string, args ...mcp.ToolOption) mcp.Tool {
	opts := []mcp.ToolOption{
		mcp.WithDescription(description),
		explainationArgument,
		environmentSourceArgument,
	}
	opts = append(opts, args...)

	return mcp.NewTool(name, opts...)
}

func newEnvironmentTool(name string, description string, args ...mcp.ToolOption) mcp.Tool {
	opts := []mcp.ToolOption{
		mcp.WithDescription(fmt.Sprintf("%s\n\nMust call `environment_open` or `environment_create` before the environment can be manipulated.", description)),
		explainationArgument,
		environmentSourceArgument,
		environmentIDArgument,
	}
	opts = append(opts, args...)

	return mcp.NewTool(name, opts...)
}
