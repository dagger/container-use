package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"dagger.io/dagger"
	"github.com/dagger/container-use/environment"
	"github.com/dagger/container-use/repository"
	"github.com/dagger/container-use/rules"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type daggerClientKey struct{}

type singleTenantKey struct{}

// single-tenant servers set this context key to indicate that this particular mcp server process will only have 1 chat session in it
// this allows api optimizations where environment_id is not required and allows claude tasks inherit their parent's envs

func openRepository(ctx context.Context, request mcp.CallToolRequest) (*repository.Repository, error) {
	// Check if we're in single-tenant mode
	singleTenant, _ := ctx.Value(singleTenantKey{}).(bool)

	var source string
	var err error

	if singleTenant {
		// In single-tenant mode, try to get from stored value first
		source = request.GetString("environment_source", "")
		if source == "" {
			source, err = getCurrentEnvironmentSource()
			if err != nil {
				return nil, err
			}
		}
	} else {
		// In multi-tenant mode, environment_source is required
		source, err = request.RequireString("environment_source")
		if err != nil {
			return nil, err
		}
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

	// Check if we're in single-tenant mode
	singleTenant, _ := ctx.Value(singleTenantKey{}).(bool)

	var envID string
	if singleTenant {
		// in single-tenant mode, environment_open requests will have environment_id. all other env-scoped tools will have "".
		envID = request.GetString("environment_id", "")
		if envID == "" {
			currentEnvID, err := getCurrentEnvironmentID()
			if err != nil {
				return nil, nil, err
			}
			envID = currentEnvID
		}
	} else {
		// In multi-tenant mode, environment_id is required
		var err error
		envID, err = request.RequireString("environment_id")
		if err != nil {
			return nil, nil, err
		}
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

func RunStdioServer(ctx context.Context, dag *dagger.Client, singleTenant bool) error {
	// Store single-tenant mode in context for tool handlers
	ctx = context.WithValue(ctx, singleTenantKey{}, singleTenant)

	s := server.NewMCPServer(
		"Dagger",
		"1.0.0",
		server.WithInstructions(rules.AgentRules),
	)

	for _, t := range createTools(singleTenant) {
		s.AddTool(t.Definition, wrapToolWithClient(t, dag, singleTenant).Handler)
	}

	slog.Info("starting server")

	stdioSrv := server.NewStdioServer(s)
	stdioSrv.SetErrorLogger(log.Default()) // this should re-use our `slog` handler

	ctx, cancel := signal.NotifyContext(ctx, getNotifySignals()...)
	defer cancel()

	err := stdioSrv.Listen(ctx, os.Stdin, os.Stdout)
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func createTools(singleTenant bool) []*Tool {
	return []*Tool{
		wrapTool(createEnvironmentOpenTool()),
		wrapTool(createEnvironmentCreateTool(singleTenant)),
		wrapTool(createEnvironmentUpdateMetadataTool(singleTenant)),
		wrapTool(createEnvironmentConfigTool(singleTenant)),
		wrapTool(createEnvironmentListTool(singleTenant)),
		wrapTool(createEnvironmentRunCmdTool(singleTenant)),
		wrapTool(createEnvironmentFileReadTool(singleTenant)),
		wrapTool(createEnvironmentFileListTool(singleTenant)),
		wrapTool(createEnvironmentFileWriteTool(singleTenant)),
		wrapTool(createEnvironmentFileEditTool(singleTenant)),
		wrapTool(createEnvironmentFileDeleteTool(singleTenant)),
		wrapTool(createEnvironmentAddServiceTool(singleTenant)),
		wrapTool(createEnvironmentCheckpointTool(singleTenant)),
	}
}

func Tools() []*Tool {
	return createTools(false) // Default to multi-tenant mode when called outside of RunStdioServer
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
func wrapToolWithClient(tool *Tool, dag *dagger.Client, singleTenant bool) *Tool {
	return &Tool{
		Definition: tool.Definition,
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ctx = context.WithValue(ctx, daggerClientKey{}, dag)
			ctx = context.WithValue(ctx, singleTenantKey{}, singleTenant)
			return tool.Handler(ctx, request)
		},
	}
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

func createEnvironmentOpenTool() *Tool {
	return &Tool{
		Definition: newEnvironmentTool(
			envToolOptions{
				name:                  "environment_open",
				description:           "Opens an existing environment. Return format is same as environment_create.",
				useCurrentEnvironment: false,
			},
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			_, env, err := openEnvironment(ctx, request)
			if err != nil {
				return nil, err
			}

			// In single-tenant mode, set this as the current environment
			if singleTenantMode, _ := ctx.Value(singleTenantKey{}).(bool); singleTenantMode {
				source, _ := request.RequireString("environment_source")
				setCurrentEnvironment(env.ID, source)
			}

			return EnvironmentToCallResult(env)
		},
	}
}

func createEnvironmentCreateTool(singleTenant bool) *Tool {
	// Build arguments dynamically based on single-tenant mode
	args := []mcp.ToolOption{
		mcp.WithString("title",
			mcp.Description("Short description of the work that is happening in this environment."),
			mcp.Required(),
		),
		mcp.WithString("from_git_ref",
			mcp.Description("Git reference to create the environment from (e.g., HEAD, main, feature-branch, SHA). Defaults to HEAD if not specified."),
		),
	}

	// Add allow_replace parameter only in single-tenant mode
	if singleTenant {
		args = append(args, mcp.WithBoolean("allow_replace",
			mcp.Description("If true and an environment already exists for this session, destructively replace it with a new one."),
		))
	}

	return &Tool{
		Definition: newRepositoryTool(
			"environment_create",
			`Creates a new development environment.
The environment is the result of a the setups commands on top of the base image.
Environment configuration is managed by the user via cu config commands.`,
			args...,
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

			// In single-tenant mode, check allow_replace before creating environment
			if singleTenantMode, _ := ctx.Value(singleTenantKey{}).(bool); singleTenantMode {
				allowReplace := request.GetBool("allow_replace", false) // Default false to prevent accidental environment replacement

				if !allowReplace {
					// Check if environment already exists
					if currentEnvID, err := getCurrentEnvironmentID(); err == nil {
						// Environment exists, return error with info about existing env
						return nil, fmt.Errorf("environment_id %s already exists for this session. Tools can be used directly. You can environment_open %s for more information, or set allow_replace=true to destructively replace it", currentEnvID, currentEnvID)
					}
				}
			}

			dag, ok := ctx.Value(daggerClientKey{}).(*dagger.Client)
			if !ok {
				return nil, fmt.Errorf("dagger client not found in context")
			}

			gitRef := request.GetString("from_git_ref", "HEAD")
			env, err := repo.Create(ctx, dag, title, request.GetString("explanation", ""), gitRef)
			if err != nil {
				return nil, fmt.Errorf("failed to create environment: %w", err)
			}

			// In single-tenant mode, set this as the current environment
			if singleTenantMode, _ := ctx.Value(singleTenantKey{}).(bool); singleTenantMode {
				source, _ := request.RequireString("environment_source")
				setCurrentEnvironment(env.ID, source)
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
}

func createEnvironmentUpdateMetadataTool(singleTenant bool) *Tool {
	return &Tool{
		Definition: newEnvironmentTool(
			envToolOptions{
				name:                  "environment_update_metadata",
				description:           "Update environment metadata such as title. This updates the descriptive information about what work is being done in the environment.",
				useCurrentEnvironment: singleTenant,
			},
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
}

func createEnvironmentConfigTool(singleTenant bool) *Tool {
	return &Tool{
		Definition: newEnvironmentTool(
			envToolOptions{
				name: "environment_config",
				description: "Make environment config changes such as base image and setup commands." +
					"If the environment is missing any tools or instructions, you MUST call this function to update the environment." +
					"You MUST update the environment with any useful tools. You will be resumed with no other context than the information provided here",
				useCurrentEnvironment: singleTenant,
			},
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
}

func createEnvironmentListTool(_ bool) *Tool {
	return &Tool{
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

			// Add warning message for LLMs
			result := string(out) + "\n\nDO NOT change environments without explicit permission from the user"
			return mcp.NewToolResultText(result), nil
		},
	}
}

func createEnvironmentRunCmdTool(singleTenant bool) *Tool {
	return &Tool{
		Definition: newEnvironmentTool(
			envToolOptions{
				name:                  "environment_run_cmd",
				description:           "Run a terminal command inside a NEW container within the environment.",
				useCurrentEnvironment: singleTenant,
			},
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

			return mcp.NewToolResultText(fmt.Sprintf("%s\n\nAny changes to the container workdir (%s) have been committed and pushed to container-use/%s remote ref", stdout, env.State.Config.Workdir, env.ID)), nil
		},
	}
}

func createEnvironmentFileReadTool(singleTenant bool) *Tool {
	return &Tool{
		Definition: newEnvironmentTool(
			envToolOptions{
				name:                  "environment_file_read",
				description:           "Read the contents of a file, specifying a line range or the entire file.",
				useCurrentEnvironment: singleTenant,
			},
			mcp.WithString("target_file",
				mcp.Description("Path of the file to read, absolute or relative to the workdir"),
				mcp.Required(),
			),
			mcp.WithBoolean("should_read_entire_file",
				mcp.Description("Whether to read the entire file. Defaults to false."),
			),
			mcp.WithNumber("start_line_one_indexed_inclusive",
				mcp.Description("The starting line (1-indexed, inclusive) to read from the file. Must specify both start_line and end_line if not reading entire file."),
			),
			mcp.WithNumber("end_line_one_indexed_inclusive",
				mcp.Description("The ending line (1-indexed, inclusive) to read from the file. Must specify both start_line and end_line if not reading entire file."),
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
}

func createEnvironmentFileListTool(singleTenant bool) *Tool {
	return &Tool{
		Definition: newEnvironmentTool(
			envToolOptions{
				name:                  "environment_file_list",
				description:           "List the contents of a directory",
				useCurrentEnvironment: singleTenant,
			},
			mcp.WithString("path",
				mcp.Description("Path of the directory to list contents of, absolute or relative to the workdir"),
				mcp.Required(),
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

			out, err := env.FileList(ctx, path)
			if err != nil {
				return nil, fmt.Errorf("failed to list directory: %w", err)
			}

			return mcp.NewToolResultText(out), nil
		},
	}
}

func createEnvironmentFileEditTool(singleTenant bool) *Tool {
	return &Tool{
		Definition: newEnvironmentTool(
			envToolOptions{
				name:                  "environment_file_edit",
				description:           "Find and replace text in a file.",
				useCurrentEnvironment: singleTenant,
			},
			mcp.WithString("target_file",
				mcp.Description("Path of the file to write, absolute or relative to the workdir."),
				mcp.Required(),
			),
			mcp.WithString("search_text",
				mcp.Description("The text to find and replace."),
				mcp.Required(),
			),
			mcp.WithString("replace_text",
				mcp.Description("The text to insert."),
				mcp.Required(),
			),
			mcp.WithString("which_match",
				mcp.Description("The ID of the match to replace, if there were multiple matches."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			repo, env, err := openEnvironment(ctx, request)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("unable to open the environment", err), nil
			}

			targetFile, err := request.RequireString("target_file")
			if err != nil {
				return nil, err
			}
			search, err := request.RequireString("search_text")
			if err != nil {
				return nil, err
			}
			replace, err := request.RequireString("replace_text")
			if err != nil {
				return nil, err
			}

			if err := env.FileEdit(ctx,
				request.GetString("explanation", ""),
				targetFile,
				search,
				replace,
				request.GetString("which_match", ""),
			); err != nil {
				return mcp.NewToolResultErrorFromErr("failed to write file", err), nil
			}

			if err := repo.UpdateFile(ctx, env, targetFile, request.GetString("explanation", "")); err != nil {
				return mcp.NewToolResultErrorFromErr("unable to update the environment", err), nil
			}

			return mcp.NewToolResultText(fmt.Sprintf("file %s edited successfully and committed to container-use/%s remote ref", targetFile, env.ID)), nil
		},
	}
}

func createEnvironmentFileWriteTool(singleTenant bool) *Tool {
	return &Tool{
		Definition: newEnvironmentTool(
			envToolOptions{
				name:                  "environment_file_write",
				description:           "Write the contents of a file.",
				useCurrentEnvironment: singleTenant,
			},
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

			if err := env.FileWrite(ctx, request.GetString("explanation", ""), targetFile, contents); err != nil {
				return nil, fmt.Errorf("failed to write file: %w", err)
			}

			if err := repo.UpdateFile(ctx, env, targetFile, request.GetString("explanation", "")); err != nil {
				return nil, fmt.Errorf("unable to update the environment: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("file %s written successfully and committed to container-use/%s remote ref", targetFile, env.ID)), nil
		},
	}
}

func createEnvironmentFileDeleteTool(singleTenant bool) *Tool {
	return &Tool{
		Definition: newEnvironmentTool(
			envToolOptions{
				name:                  "environment_file_delete",
				description:           "Deletes a file at the specified path.",
				useCurrentEnvironment: singleTenant,
			},
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

			if err := env.FileDelete(ctx, request.GetString("explanation", ""), targetFile); err != nil {
				return nil, fmt.Errorf("failed to delete file: %w", err)
			}

			if err := repo.Update(ctx, env, request.GetString("explanation", "")); err != nil {
				return nil, fmt.Errorf("failed to update env: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("file %s deleted successfully and committed to container-use/%s remote ref", targetFile, env.ID)), nil
		},
	}
}

func createEnvironmentCheckpointTool(singleTenant bool) *Tool {
	return &Tool{
		Definition: newEnvironmentTool(
			envToolOptions{
				name:                  "environment_checkpoint",
				description:           "Checkpoints an environment in its current state as a container.",
				useCurrentEnvironment: singleTenant,
			},
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
}

func createEnvironmentAddServiceTool(singleTenant bool) *Tool {
	return &Tool{
		Definition: newEnvironmentTool(
			envToolOptions{
				name:                  "environment_add_service",
				description:           "Add a service to the environment (e.g. database, cache, etc.)",
				useCurrentEnvironment: singleTenant,
			},
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

			service, err := env.AddService(ctx, request.GetString("explanation", ""), &environment.ServiceConfig{
				Name:         serviceName,
				Image:        image,
				Command:      command,
				ExposedPorts: ports,
				Env:          envs,
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
}
