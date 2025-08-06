package mcpserver

import "github.com/mark3labs/mcp-go/mcp"

var (
	explanationArgument = mcp.WithString("explanation",
		mcp.Description("One sentence explanation for why this tool is being called."),
	)
	environmentSourceArgument = mcp.WithString("environment_source",
		mcp.Description("Absolute path to the source git repository for the environment."),
		mcp.Required(),
	)
	environmentIDArgument = mcp.WithString("environment_id",
		mcp.Description("The UUID of the environment for this command."),
		mcp.Required(),
	)
)

func newRepositoryTool(name string, description string, args ...mcp.ToolOption) mcp.Tool {
	opts := []mcp.ToolOption{
		mcp.WithDescription(description),
		explanationArgument,
		environmentSourceArgument,
	}

	opts = append(opts, args...)
	return mcp.NewTool(name, opts...)
}

type envToolOptions struct {
	name                  string
	description           string
	useCurrentEnvironment bool
}

func newEnvironmentTool(toolOptions envToolOptions, mcpToolOptions ...mcp.ToolOption) mcp.Tool {
	opts := []mcp.ToolOption{
		mcp.WithDescription(toolOptions.description),
		explanationArgument,
	}

	// in single-tenant mode, environment tools (except open) use currentEnvironmentID & currentEnvironmentSource as their target env
	if !toolOptions.useCurrentEnvironment {
		opts = append(opts, environmentSourceArgument)
		opts = append(opts, environmentIDArgument)
	}

	opts = append(opts, mcpToolOptions...)
	return mcp.NewTool(toolOptions.name, opts...)
}
