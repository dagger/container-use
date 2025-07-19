package mcpserver

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"dagger.io/dagger"
	"github.com/dagger/container-use/environment"
	"github.com/dagger/container-use/repository"
	"github.com/dagger/container-use/rules"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type daggerClientKey struct{}

func openRepository(ctx context.Context, request mcp.CallToolRequest) (*repository.Repository, error) {
	source, err := request.RequireString("environment_source")
	if err != nil {
		return nil, err
	}
	repo, err := repository.Open(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("unable to open repository: %w", err)
	}
	return repo, nil
}

func openEnvironment(ctx context.Context, request mcp.CallToolRequest) (*repository.Repository, *environment.Environment, error) {
	repo, err := openRepository(ctx, request)
	if err != nil {
		return nil, nil, err
	}
	envID, err := request.RequireString("environment_id")
	if err != nil {
		return nil, nil, err
	}
	dag, ok := ctx.Value(daggerClientKey{}).(*dagger.Client)
	if !ok {
		return nil, nil, fmt.Errorf("dagger client not found in context")
	}
	env, err := repo.Get(ctx, dag, envID)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get environment: %w", err)
	}
	return repo, env, nil
}

type Tool struct {
	Definition mcp.Tool
	Handler    server.ToolHandlerFunc
}

func RunStdioServer(ctx context.Context, dag *dagger.Client) error {
	s := server.NewMCPServer(
		"Dagger",
		"1.0.0",
		server.WithInstructions(rules.AgentRules),
	)

	for _, t := range tools {
		s.AddTool(t.Definition, wrapToolWithClient(t, dag).Handler)
	}

	slog.Info("starting server")

	stdioSrv := server.NewStdioServer(s)
	stdioSrv.SetErrorLogger(log.Default()) // this should re-use our `slog` handler

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill, syscall.SIGTERM)
	defer cancel()

	err := stdioSrv.Listen(ctx, os.Stdin, os.Stdout)
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

var tools = []*Tool{}

func Tools() []*Tool {
	return tools
}

func registerTool(tool ...*Tool) {
	for _, t := range tool {
		tools = append(tools, wrapTool(t))
	}
}

func wrapTool(tool *Tool) *Tool {
	return &Tool{
		Definition: tool.Definition,
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			slog.Info("Tool called", "tool", tool.Definition.Name)
			defer func() {
				slog.Info("Tool finished", "tool", tool.Definition.Name)
			}()
			response, err := tool.Handler(ctx, request)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return response, nil
		},
	}
}

// keeping this modular for now. we could move tool registration to RunStdioServer and collapse the 2 wrapTool functions.
func wrapToolWithClient(tool *Tool, dag *dagger.Client) *Tool {
	return &Tool{
		Definition: tool.Definition,
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ctx = context.WithValue(ctx, daggerClientKey{}, dag)
			return tool.Handler(ctx, request)
		},
	}
}

func init() {
	registerTool(
		EnvironmentOpenTool,
		EnvironmentCreateTool,
		EnvironmentUpdateMetadataTool,
		EnvironmentConfigTool,

		EnvironmentRunCmdTool,

		EnvironmentFileListTool,
		EnvironmentFileGlobTool,
		EnvironmentFileGrepTool,
		EnvironmentFileReadTool,
		EnvironmentFileWriteTool,
		EnvironmentFileEditTool,
		EnvironmentFileDeleteTool,

		EnvironmentAddServiceTool,

		EnvironmentCheckpointTool,
	)
}

type EnvironmentResponse struct {
	ID              string                         `json:"id"`
	Title           string                         `json:"title"`
	Config          *environment.EnvironmentConfig `json:"config"`
	RemoteRef       string                         `json:"remote_ref"`
	CheckoutCommand string                         `json:"checkout_command_to_share_with_user"`
	LogCommand      string                         `json:"log_command_to_share_with_user"`
	DiffCommand     string                         `json:"diff_command_to_share_with_user"`
	Services        []*environment.Service         `json:"services,omitempty"`
}

func environmentResponseFromEnvInfo(envInfo *environment.EnvironmentInfo) *EnvironmentResponse {
	return &EnvironmentResponse{
		ID:              envInfo.ID,
		Title:           envInfo.State.Title,
		Config:          envInfo.State.Config,
		RemoteRef:       fmt.Sprintf("container-use/%s", envInfo.ID),
		CheckoutCommand: fmt.Sprintf("container-use checkout %s", envInfo.ID),
		LogCommand:      fmt.Sprintf("container-use log %s", envInfo.ID),
		DiffCommand:     fmt.Sprintf("container-use diff %s", envInfo.ID),
		Services:        nil, // EnvironmentInfo doesn't have "active" services, specifically useful for EndpointMappings
	}
}

func environmentResponseFromEnv(env *environment.Environment) *EnvironmentResponse {
	resp := environmentResponseFromEnvInfo(env.EnvironmentInfo)
	resp.Services = env.Services
	return resp
}

func marshalEnvironment(env *environment.Environment) (string, error) {
	out, err := json.Marshal(environmentResponseFromEnv(env))
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(out), nil
}

func marshalEnvironmentInfo(envInfo *environment.EnvironmentInfo) (string, error) {
	out, err := json.Marshal(environmentResponseFromEnvInfo(envInfo))
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(out), nil
}

func EnvironmentToCallResult(env *environment.Environment) (*mcp.CallToolResult, error) {
	out, err := marshalEnvironment(env)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(out), nil
}

func EnvironmentInfoToCallResult(envInfo *environment.EnvironmentInfo) (*mcp.CallToolResult, error) {
	out, err := marshalEnvironmentInfo(envInfo)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(out), nil
}

var EnvironmentOpenTool = &Tool{
	Definition: newEnvironmentTool(
		"environment_open",
		"Opens an existing environment. Return format is same as environment_create.",
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_, env, err := openEnvironment(ctx, request)
		if err != nil {
			return nil, err
		}
		return EnvironmentToCallResult(env)
	},
}

var EnvironmentCreateTool = &Tool{
	Definition: newRepositoryTool(
		"environment_create",
		`Creates a new development environment.
The environment is the result of a the setups commands on top of the base image.
Environment configuration is managed by the user via cu config commands.`,
		mcp.WithString("title",
			mcp.Description("Short description of the work that is happening in this environment."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		repo, err := openRepository(ctx, request)
		if err != nil {
			return nil, err
		}
		title, err := request.RequireString("title")
		if err != nil {
			return nil, err
		}

		dag, ok := ctx.Value(daggerClientKey{}).(*dagger.Client)
		if !ok {
			return nil, fmt.Errorf("dagger client not found in context")
		}

		env, err := repo.Create(ctx, dag, title, request.GetString("explanation", ""))
		if err != nil {
			return nil, fmt.Errorf("failed to create environment: %w", err)
		}

		out, err := marshalEnvironment(env)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal environment: %w", err)
		}

		dirty, status, err := repo.IsDirty(ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to check if environment is dirty: %w", err)
		}

		if !dirty {
			return mcp.NewToolResultText(out), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf(`%s

CRITICAL: You MUST inform the user that the repository %s has uncommitted changes that are NOT included in this environment. The environment was created from the last committed state only.

Uncommitted changes detected:
%s

You MUST tell the user: To include these changes in the environment, they need to commit them first using git commands outside the environment.`, out, request.GetString("environment_source", ""), status)), nil
	},
}

var EnvironmentUpdateMetadataTool = &Tool{
	Definition: newEnvironmentTool(
		"environment_update_metadata",
		"Update environment metadata such as title. This updates the descriptive information about what work is being done in the environment.",
		mcp.WithString("title",
			mcp.Description("Updated title describing the work being done in this environment."),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		repo, env, err := openEnvironment(ctx, request)
		if err != nil {
			return nil, err
		}

		// Update title if provided
		if title := request.GetString("title", ""); title != "" {
			env.State.Title = title
		}

		if err := repo.Update(ctx, env, request.GetString("explanation", "")); err != nil {
			return nil, fmt.Errorf("unable to update the environment: %w", err)
		}

		out, err := marshalEnvironment(env)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal environment: %w", err)
		}
		return mcp.NewToolResultText(fmt.Sprintf("Environment metadata updated successfully.\n%s", out)), nil
	},
}

var EnvironmentConfigTool = &Tool{
	Definition: newEnvironmentTool(
		"environment_config",
		"Make environment config changes such as base image and setup commands."+
			"If the environment is missing any tools or instructions, you MUST call this function to update the environment."+
			"You MUST update the environment with any useful tools. You will be resumed with no other context than the information provided here",
		mcp.WithObject("config",
			mcp.Required(),
			mcp.Properties(map[string]any{
				"base_image": map[string]any{
					"type":        "string",
					"description": "Base image for the environment",
				},
				"setup_commands": map[string]any{
					"type":        "array",
					"description": "Commands that should be executed on top of the base image to set up the environment. Similar to `RUN` instructions in Dockerfiles.",
					"items":       map[string]any{"type": "string"},
				},
				"envs": map[string]any{
					"type":        "array",
					"description": "The environment variables to set (e.g. `[\"FOO=bar\", \"BAZ=qux\"]`).",
					"items":       map[string]any{"type": "string"},
				},
			}),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		repo, env, err := openEnvironment(ctx, request)
		if err != nil {
			return nil, err
		}

		updatedConfig := env.State.Config.Copy()

		newConfig, ok := request.GetArguments()["config"].(map[string]any)
		if !ok {
			return nil, errors.New("invalid config")
		}

		if baseImage, ok := newConfig["base_image"].(string); ok {
			updatedConfig.BaseImage = baseImage
		}

		if setupCommands, ok := newConfig["setup_commands"].([]any); ok {
			updatedConfig.SetupCommands = make([]string, len(setupCommands))
			for i, command := range setupCommands {
				updatedConfig.SetupCommands[i] = command.(string)
			}
		}

		if envs, ok := newConfig["envs"].([]any); ok {
			updatedConfig.Env = make([]string, len(envs))
			for i, env := range envs {
				updatedConfig.Env[i] = env.(string)
			}
		}

		if err := env.UpdateConfig(ctx, updatedConfig); err != nil {
			return nil, fmt.Errorf("unable to update the environment: %w", err)
		}

		if err := repo.Update(ctx, env, request.GetString("explanation", "")); err != nil {
			return nil, fmt.Errorf("failed to update repository: %w", err)
		}

		out, err := marshalEnvironment(env)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal environment: %w", err)
		}

		message := fmt.Sprintf(`SUCCESS: Configuration successfully applied. Environment has been restarted, all previous commands have been lost.
IMPORTANT: The configuration changes are LOCAL to this environment.
TELL THE USER: To make these changes persistent, they will have to run "cu config import %s"

%s
`, env.ID, out)

		return mcp.NewToolResultText(message), nil
	},
}

var EnvironmentListTool = &Tool{
	Definition: newRepositoryTool(
		"environment_list",
		"List available environments",
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		repo, err := openRepository(ctx, request)
		if err != nil {
			return nil, err
		}
		envInfos, err := repo.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("invalid source: %w", err)
		}

		// Convert EnvironmentInfo slice to EnvironmentResponse slice
		responses := make([]EnvironmentResponse, len(envInfos))
		for i, envInfo := range envInfos {
			responses[i] = *environmentResponseFromEnvInfo(envInfo)
		}

		out, err := json.Marshal(responses)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	},
}

var EnvironmentRunCmdTool = &Tool{
	Definition: newEnvironmentTool(
		"environment_run_cmd",
		"Run a terminal command inside a NEW container within the environment.",
		mcp.WithString("command",
			mcp.Description("The terminal command to execute. If empty, the environment's default command is used."),
		),
		mcp.WithString("shell",
			mcp.Description("The shell that will be interpreting this command (default: sh)"),
		),
		mcp.WithBoolean("background",
			mcp.Description(`Run the command in the background
Must ALWAYS be set for long running command (e.g. http server).
Failure to do so will result in the tool being stuck, awaiting for the command to finish.`,
			),
		),
		mcp.WithBoolean("use_entrypoint",
			mcp.Description("Use the image entrypoint, if present, by prepending it to the args."),
		),
		mcp.WithArray("ports",
			mcp.Description("Ports to expose. Only works with background environments. For each port, returns the environment_internal (for use inside environments) and host_external (for use by the user) addresses."),
			mcp.Items(map[string]any{"type": "number"}),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		repo, env, err := openEnvironment(ctx, request)
		if err != nil {
			return nil, err
		}

		command := request.GetString("command", "")
		shell := request.GetString("shell", "sh")

		updateRepo := func() error {
			if err := repo.Update(ctx, env, request.GetString("explanation", "")); err != nil {
				return fmt.Errorf("failed to update repository: %w", err)
			}
			return nil
		}

		background := request.GetBool("background", false)
		if background {
			ports := []int{}
			if portList, ok := request.GetArguments()["ports"].([]any); ok {
				for _, port := range portList {
					ports = append(ports, int(port.(float64)))
				}
			}
			endpoints, runErr := env.RunBackground(ctx, command, shell, ports, request.GetBool("use_entrypoint", false))
			// We want to update the repository even if the command failed.
			if err := updateRepo(); err != nil {
				return nil, err
			}
			if runErr != nil {
				return nil, fmt.Errorf("failed to run command: %w", runErr)
			}

			out, err := json.Marshal(endpoints)
			if err != nil {
				return nil, err
			}

			return mcp.NewToolResultText(fmt.Sprintf(`Command started in the background in NEW container. Endpoints are %s

To access from the user's machine: use host_external. To access from other commands in this environment: use environment_internal.

Any changes to the container workdir (%s) WILL NOT be committed to container-use/%s

Background commands are unaffected by filesystem and any other kind of changes. You need to start a new command for changes to take effect.`,
				string(out), env.State.Config.Workdir, env.ID)), nil
		}

		stdout, runErr := env.Run(ctx, command, shell, request.GetBool("use_entrypoint", false))
		// We want to update the repository even if the command failed.
		if err := updateRepo(); err != nil {
			return nil, err
		}
		if runErr != nil {
			return nil, fmt.Errorf("failed to run command: %w", runErr)
		}

		return mcp.NewToolResultText(fmt.Sprintf("%s\n\nAny changes to the container workdir (%s) have been committed and pushed to container-use/ remote", stdout, env.State.Config.Workdir)), nil
	},
}

var EnvironmentFileReadTool = &Tool{
	Definition: newEnvironmentTool(
		"environment_file_read",
		"Read the contents of a file, specifying a line range or the entire file.",
		mcp.WithString("target_file",
			mcp.Description("Path of the file to read, absolute or relative to the workdir"),
			mcp.Required(),
		),
		mcp.WithBoolean("should_read_entire_file",
			mcp.Description("Whether to read the entire file. Defaults to false."),
		),
		mcp.WithNumber("start_line_one_indexed_inclusive",
			mcp.Description("The one-indexed line number to start reading from (inclusive)."),
		),
		mcp.WithNumber("end_line_one_indexed_inclusive",
			mcp.Description("The one-indexed line number to end reading at (inclusive)."),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_, env, err := openEnvironment(ctx, request)
		if err != nil {
			return nil, err
		}

		targetFile, err := request.RequireString("target_file")
		if err != nil {
			return nil, err
		}
		shouldReadEntireFile := request.GetBool("should_read_entire_file", false)
		startLineOneIndexedInclusive := request.GetInt("start_line_one_indexed_inclusive", 0)
		endLineOneIndexedInclusive := request.GetInt("end_line_one_indexed_inclusive", 0)

		fileContents, err := env.FileRead(ctx, targetFile, shouldReadEntireFile, startLineOneIndexedInclusive, endLineOneIndexedInclusive)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		return mcp.NewToolResultText(fileContents), nil
	},
}

var EnvironmentFileListTool = &Tool{
	Definition: newEnvironmentTool(
		"environment_file_list",
		"List files and directories in a given path. You can optionally provide an array of glob patterns to ignore with the ignore parameter. You should generally prefer the environment_file_glob and environment_file_grep tools, if you know which directories to search.",
		mcp.WithString("path",
			mcp.Description("Path of the directory to list contents of, absolute or relative to the workdir"),
			mcp.Required(),
		),
		mcp.WithArray("ignore",
			mcp.Description("List of glob patterns to ignore"),
			mcp.Items(map[string]any{"type": "string"}),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_, env, err := openEnvironment(ctx, request)
		if err != nil {
			return nil, err
		}

		path, err := request.RequireString("path")
		if err != nil {
			return nil, err
		}
		args := request.GetArguments()
		v, ok := args["ignore"]
		var ignore []string
		if ok {
			ignore, ok = v.([]string)
			if !ok {
				return nil, fmt.Errorf("`ignore` argument expects an array of strings")
			}
		}
		out, err := env.FileList(ctx, path, ignore)
		if err != nil {
			return nil, fmt.Errorf("failed to list directory: %w", err)
		}

		return mcp.NewToolResultText(out), nil
	},
}

var EnvironmentFileWriteTool = &Tool{
	Definition: newEnvironmentTool(
		"environment_file_write",
		"Write the full contents of a file.",
		mcp.WithString("target_file",
			mcp.Description("Path of the file to write, absolute or relative to the workdir."),
			mcp.Required(),
		),
		mcp.WithString("contents",
			mcp.Description("Full text content of the file you want to write."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		repo, env, err := openEnvironment(ctx, request)
		if err != nil {
			return nil, err
		}

		targetFile, err := request.RequireString("target_file")
		if err != nil {
			return nil, err
		}
		contents, err := request.RequireString("contents")
		if err != nil {
			return nil, err
		}

		if err := env.FileWrite(ctx, targetFile, contents); err != nil {
			return nil, fmt.Errorf("failed to write file: %w", err)
		}

		if err := repo.Update(ctx, env, request.GetString("explanation", "")); err != nil {
			return nil, fmt.Errorf("unable to update the environment: %w", err)
		}

		return mcp.NewToolResultText(fmt.Sprintf("file %s written successfully and committed to container-use/ remote", targetFile)), nil
	},
}

var EnvironmentFileGlobTool = &Tool{
	Definition: newEnvironmentTool(
		"environment_file_glob",
		"Fast file pattern matching tool.\nSupports glob syntax \"**/*.js\" or \"src/**/*.ts\".\nReturns matching file paths.",
		mcp.WithString("path",
			mcp.Description("Path of the directory to search in, absolute or relative to the workdir"),
			mcp.Required(),
		),
		mcp.WithString("pattern",
			mcp.Description("The glob pattern to match file paths against."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_, env, err := openEnvironment(ctx, request)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("unable to open the environment", err), nil
		}

		path, err := request.RequireString("path")
		if err != nil {
			return nil, err
		}

		pattern, err := request.RequireString("pattern")
		if err != nil {
			return nil, err
		}

		out, err := env.FileGlob(ctx, path, pattern)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to list directory", err), nil
		}

		return mcp.NewToolResultText(out), nil
	},
}

var EnvironmentFileGrepTool = &Tool{
	Definition: newEnvironmentTool(
		"environment_file_grep",
		"Fast file content search using regular expressions.\nSupports full regex syntax (eg. \"log.*Error\", \"function\\s+\\w+\", etc.).\nReturns matching file paths.",
		mcp.WithString("path",
			mcp.Description("Path of the directory to search in, absolute or relative to the workdir"),
			mcp.Required(),
		),
		mcp.WithString("pattern",
			mcp.Description("The regular expression pattern to search for in file contents."),
			mcp.Required(),
		),
		mcp.WithString("include",
			mcp.Description("File pattern to include in the search (e.g. \"*.js\", \"*.{ts,tsx}\")."),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_, env, err := openEnvironment(ctx, request)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("unable to open the environment", err), nil
		}

		path, err := request.RequireString("path")
		if err != nil {
			return nil, err
		}

		pattern, err := request.RequireString("pattern")
		if err != nil {
			return nil, err
		}

		include := request.GetString("include", "")

		out, err := env.FileGrep(ctx, path, pattern, include)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to grep directory", err), nil
		}
		if out == "" {
			out = "No files found matching the pattern."
		}

		return mcp.NewToolResultText(out), nil
	},
}

var EnvironmentFileEditTool = &Tool{
	Definition: newEnvironmentTool(
		"environment_file_edit",
		"This is a tool for making single or multiple edits to a single file in one operation. It allows you to perform multiple find-and-replace operations efficiently.\n\nBefore using this tool:\n\n1. Use the environment_file_read tool to understand the file's contents and context\n2. Verify the directory path is correct\n\nTo make file edits, provide the following:\n1. file_path: The relative path to the file to modify (must be relative, not absolute)\n2. edits: An array of edit operations to perform, where each edit contains:\n   - old_string: The text to replace (must match the file contents exactly, including all whitespace and indentation)\n   - new_string: The edited text to replace the old_string\n   - replace_all: Replace all occurences of old_string. This parameter is optional and defaults to false.\n\nIMPORTANT:\n- All edits are applied in sequence, in the order they are provided\n- Each edit operates on the result of the previous edit\n- All edits must be valid for the operation to succeed - if any edit fails, none will be applied\n- This tool is ideal when you need to make several changes to different parts of the same file\n- For Jupyter notebooks (.ipynb files), use the NotebookEdit instead\n\nCRITICAL REQUIREMENTS:\n1. All edits follow the same requirements as the single Edit tool\n2. The edits are atomic - either all succeed or none are applied\n3. Plan your edits carefully to avoid conflicts between sequential operations\n\nWARNING:\n- The tool will fail if edits.old_string doesn't match the file contents exactly (including whitespace)\n- The tool will fail if edits.old_string and edits.new_string are the same\n- Since edits are applied in sequence, ensure that earlier edits don't affect the text that later edits are trying to find\n\nWhen making edits:\n- Ensure all edits result in idiomatic, correct code\n- Do not leave the code in a broken state\n- Always use absolute file paths (starting with /)\n- Only use emojis if the user explicitly requests it. Avoid adding emojis to files unless asked.\n- Use replace_all for replacing and renaming strings across the file. This parameter is useful if you want to rename a variable for instance.\n\nIf you want to create a new file, use:\n- A new file path, including dir name if needed\n- First edit: empty old_string and the new file's contents as new_string\n- Subsequent edits: normal edit operations on the created content",
		mcp.WithString("target_file",
			mcp.Description("Path of the file to edit, absolute or relative to the workdir."),
			mcp.Required(),
		),
		mcp.WithArray("edits",
			mcp.Description("An array of edit operations to perform on the contents of target_file."),
			mcp.Items(map[string]any{"type": "object", "properties": map[string]any{
				"old_string":  map[string]any{"type": "string", "description": "The text to replace"},
				"new_string":  map[string]any{"type": "string", "description": "The text to replace it with"},
				"replace_all": map[string]any{"type": "string", "description": "Replace all occurences of old_string (default false)", "default": false},
			}}),
			mcp.MinItems(1),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		repo, env, err := openEnvironment(ctx, request)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("unable to open the environment", err), nil
		}

		var args struct {
			TargetFile string
			Edits      []environment.FileEdit
		}
		if err := request.BindArguments(&args); err != nil {
			return nil, fmt.Errorf("could not bind arguments")
		}

		if err := env.FileEdit(ctx, args.TargetFile, args.Edits); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to edit file", err), nil
		}

		if err := repo.Update(ctx, env, request.GetString("explanation", "")); err != nil {
			return mcp.NewToolResultErrorFromErr("unable to update the environment", err), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("file %s edited successfully and committed to container-use/ remote", args.TargetFile)), nil
	},
}

var EnvironmentFileDeleteTool = &Tool{
	Definition: newEnvironmentTool(
		"environment_file_delete",
		"Deletes a file at the specified path.",
		mcp.WithString("target_file",
			mcp.Description("Path of the file to delete, absolute or relative to the workdir."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		repo, env, err := openEnvironment(ctx, request)
		if err != nil {
			return nil, err
		}

		targetFile, err := request.RequireString("target_file")
		if err != nil {
			return nil, err
		}

		if err := env.FileDelete(ctx, targetFile); err != nil {
			return nil, fmt.Errorf("failed to delete file: %w", err)
		}

		if err := repo.Update(ctx, env, request.GetString("explanation", "")); err != nil {
			return nil, fmt.Errorf("failed to update env: %w", err)
		}

		return mcp.NewToolResultText(fmt.Sprintf("file %s deleted successfully and committed to container-use/ remote", targetFile)), nil
	},
}

var EnvironmentCheckpointTool = &Tool{
	Definition: newEnvironmentTool(
		"environment_checkpoint",
		"Checkpoints an environment in its current state as a container.",
		mcp.WithString("destination",
			mcp.Description("Container image destination to checkpoint to (e.g. registry.com/user/image:tag"),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_, env, err := openEnvironment(ctx, request)
		if err != nil {
			return nil, err
		}
		destination, err := request.RequireString("destination")
		if err != nil {
			return nil, err
		}

		endpoint, err := env.Checkpoint(ctx, destination)
		if err != nil {
			return nil, fmt.Errorf("failed to checkpoint environment: %w", err)
		}
		return mcp.NewToolResultText(fmt.Sprintf("Checkpoint pushed to %q. You MUST use the full content addressed (@sha256:...) reference in `docker` commands. The entrypoint is set to `sh`, keep that in mind when giving commands to the container.", endpoint)), nil
	},
}

var EnvironmentAddServiceTool = &Tool{
	Definition: newEnvironmentTool(
		"environment_add_service",
		"Add a service to the environment (e.g. database, cache, etc.)",
		mcp.WithString("name",
			mcp.Description("The name of the service to start."),
			mcp.Required(),
		),
		mcp.WithString("image",
			mcp.Description("The image of the service to start."),
			mcp.Required(),
		),
		mcp.WithString("command",
			mcp.Description("The command to start the service. If not provided the image default command will be used."),
		),
		mcp.WithArray("ports",
			mcp.Description("Ports to expose. For each port, returns the container_internal (for use by environments) and host_external (for use by the user) address."),
			mcp.Items(map[string]any{"type": "number"}),
		),
		mcp.WithArray("envs",
			mcp.Description("The environment variables to set (e.g. `[\"FOO=bar\", \"BAZ=qux\"]`)."),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithArray("secrets",
			mcp.Description(`Secret references in the format of "SECRET_NAME=schema://value

Secrets will be available in the environment as environment variables ($SECRET_NAME).

Supported schemas are:
- file://PATH: local file path
- env://NAME: environment variable
- op://<vault-name>/<item-name>/[section-name/]<field-name>: 1Password secret
`),
			mcp.Items(map[string]any{"type": "string"}),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		repo, env, err := openEnvironment(ctx, request)
		if err != nil {
			return nil, err
		}
		serviceName, err := request.RequireString("name")
		if err != nil {
			return nil, err
		}
		image, err := request.RequireString("image")
		if err != nil {
			return nil, err
		}
		command := request.GetString("command", "")
		ports := []int{}
		if portList, ok := request.GetArguments()["ports"].([]any); ok {
			for _, port := range portList {
				ports = append(ports, int(port.(float64)))
			}
		}

		envs := request.GetStringSlice("envs", []string{})
		secrets := request.GetStringSlice("secrets", []string{})

		service, err := env.AddService(ctx, request.GetString("explanation", ""), &environment.ServiceConfig{
			Name:         serviceName,
			Image:        image,
			Command:      command,
			ExposedPorts: ports,
			Env:          envs,
			Secrets:      secrets,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add service: %w", err)
		}

		if err := repo.Update(ctx, env, request.GetString("explanation", "")); err != nil {
			return nil, fmt.Errorf("failed to update env: %w", err)
		}

		output, err := json.Marshal(service)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal service: %w", err)
		}

		return mcp.NewToolResultText(fmt.Sprintf("Service added and started successfully: %s", string(output))), nil
	},
}
