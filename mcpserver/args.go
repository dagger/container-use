package mcpserver

import "github.com/mark3labs/mcp-go/mcp"

var (
	explanationArgument = mcp.WithString("explanation",
		mcp.Description("One sentence explanation for why this directory is being listed."),
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

func newRepositoryTool(name string, description string, singleTenant bool, args ...mcp.ToolOption) mcp.Tool {
	opts := []mcp.ToolOption{
		mcp.WithDescription(description),
		explanationArgument,
	}

	// Only include environment_source for multi-tenant mode or create/open operations
	if !singleTenant {
		opts = append(opts, environmentSourceArgument)
	}

	opts = append(opts, args...)
	return mcp.NewTool(name, opts...)
}

func newEnvironmentTool(name string, description string, includeEnvIDArg bool, singleTenant bool, args ...mcp.ToolOption) mcp.Tool {
	opts := []mcp.ToolOption{
		mcp.WithDescription(description),
		explanationArgument,
	}

	// Only include environment_source for multi-tenant mode or create/open operations
	if !singleTenant {
		opts = append(opts, environmentSourceArgument)
	}

	if includeEnvIDArg {
		opts = append(opts, environmentIDArgument)
	}

	opts = append(opts, args...)
	return mcp.NewTool(name, opts...)
}
