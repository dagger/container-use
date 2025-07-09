package mcpserver

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"

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
		return nil, err
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
		return nil, nil, err
	}
	return repo, env, nil
}

type Tool[R Request, V any] struct {
	Definition    mcp.Tool
	ParseRequest  func(request mcp.CallToolRequest) (*R, error)
	ActualHandler func(ctx context.Context, request *R) (*ToolResponse[V], error)
	Handler       server.ToolHandlerFunc
}

type MCPHandler struct {
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

	return stdioSrv.Listen(ctx, os.Stdin, os.Stdout)
}

var tools = []MCPHandler{}

func registerTool[R Request, V Response](tool *Tool[R, V]) {
	tools = append(tools, createHandler(tool))
}

func createHandler[R Request, V Response](tool *Tool[R, V]) MCPHandler {
	return MCPHandler{
		Definition: tool.Definition,
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			slog.Info("Tool called", "tool", tool.Definition.Name)
			defer func() {
				slog.Info("Tool finished", "tool", tool.Definition.Name)
			}()
			// FIXME(aluzzardi): Backward compat, remove this
			if tool.Handler != nil {
				return tool.Handler(ctx, request)
			}
			parsed, err := tool.ParseRequest(request)
			if err != nil {
				return nil, err
			}
			response, err := tool.ActualHandler(ctx, parsed)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			json, err := json.Marshal(response.Data)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("failed to marshal response", err), nil
			}

			body := string(json)

			if response.Message != "" {
				body += "\n" + response.Message
			}

			return mcp.NewToolResultText(body), nil
		},
	}
}

// keeping this modular for now. we could move tool registration to RunStdioServer and collapse the 2 wrapTool functions.
func wrapToolWithClient(tool MCPHandler, dag *dagger.Client) MCPHandler {
	return MCPHandler{
		Definition: tool.Definition,
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ctx = context.WithValue(ctx, daggerClientKey{}, dag)
			return tool.Handler(ctx, request)
		},
	}
}

func init() {
	registerTool(EnvironmentOpenTool)
	registerTool(EnvironmentCreateTool)
	registerTool(EnvironmentUpdateTool)

	registerTool(EnvironmentRunCmdTool)

	registerTool(EnvironmentFileReadTool)
	registerTool(EnvironmentFileListTool)
	registerTool(EnvironmentFileWriteTool)
	registerTool(EnvironmentFileDeleteTool)

	registerTool(EnvironmentAddServiceTool)
	registerTool(EnvironmentCheckpointTool)
}

type EnvironmentResponse struct {
	ID              string                 `json:"id"`
	Title           string                 `json:"title"`
	BaseImage       string                 `json:"base_image"`
	SetupCommands   []string               `json:"setup_commands"`
	Instructions    string                 `json:"instructions"`
	Workdir         string                 `json:"workdir"`
	RemoteRef       string                 `json:"remote_ref"`
	CheckoutCommand string                 `json:"checkout_command_to_share_with_user"`
	LogCommand      string                 `json:"log_command_to_share_with_user"`
	DiffCommand     string                 `json:"diff_command_to_share_with_user"`
	Services        []*environment.Service `json:"services,omitempty"`
}

func environmentResponseFromEnvInfo(envInfo *environment.EnvironmentInfo) *EnvironmentResponse {
	return &EnvironmentResponse{
		ID:              envInfo.ID,
		Title:           envInfo.State.Title,
		Instructions:    envInfo.Config.Instructions,
		BaseImage:       envInfo.Config.BaseImage,
		SetupCommands:   envInfo.Config.SetupCommands,
		Workdir:         envInfo.Config.Workdir,
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

var EnvironmentOpenTool = &Tool[Request, Response]{
	Definition: mcp.NewTool("environment_open",
		mcp.WithDescription("Opens an existing environment. Return format is same as environment_create."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this environment is being opened."),
		),
		mcp.WithString("environment_source",
			mcp.Description("Path to the source git repository for the environment. Prefer absolute paths from available context, but if you don't have one, you can try '.'"),
			mcp.Required(),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment to open."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_, env, err := openEnvironment(ctx, request)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("unable to open the environment", err), nil
		}
		return EnvironmentToCallResult(env)
	},
}

type EnvironmentCreateToolRequest struct {
	BaseRepositoryRequest

	Title string `json:"title"`
}

var EnvironmentCreateTool = &Tool[EnvironmentCreateToolRequest, *environment.Environment]{
	Definition: newRepositoryTool(
		"environment_create",
		`Creates a new development environment.
	The environment is the result of a the setups commands on top of the base image.
	Read carefully the instructions to understand the environment.
	DO NOT manually install toolchains inside the environment, instead explicitly call environment_update`,
		mcp.WithString("title",
			mcp.Description("Short description of the work that is happening in this environment. Keep this title updated using `environment_update`."),
			mcp.Required(),
		),
	),
	ParseRequest: func(request mcp.CallToolRequest) (*EnvironmentCreateToolRequest, error) {
		base, err := parseBaseRepositoryRequest(request)
		if err != nil {
			return nil, err
		}
		return &EnvironmentCreateToolRequest{
			BaseRepositoryRequest: base,
			Title:                 request.GetString("title", ""),
		}, nil
	},
	ActualHandler: func(ctx context.Context, request *EnvironmentCreateToolRequest) (*ToolResponse[*environment.Environment], error) {
		repo, err := openRepositoryFromRequest(ctx, request.BaseRepositoryRequest)
		if err != nil {
			return nil, fmt.Errorf("unable to open the repository: %w", err)
		}

		env, err := repo.Create(ctx, dagFromContext(ctx), request.Title, request.Explanation)
		if err != nil {
			return nil, fmt.Errorf("failed to create environment: %w", err)
		}

		resp := &ToolResponse[*environment.Environment]{
			Data: env,
		}

		dirty, status, err := repo.IsDirty(ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to check if environment is dirty: %w", err)
		}

		if dirty {
			resp.Message = fmt.Sprintf(`
CRITICAL: You MUST inform the user that the repository %s has uncommitted changes that are NOT included in this environment. The environment was created from the last committed state only.

Uncommitted changes detected:
%s

You MUST tell the user: To include these changes in the environment, they need to commit them first using git commands outside the environment.`,
				request.EnvironmentSource, status)
		}
		return resp, nil
	},
}

var EnvironmentUpdateTool = &Tool[Request, Response]{
	Definition: mcp.NewTool("environment_update",
		mcp.WithDescription("Updates an environment with new instructions and toolchains."+
			"If the environment is missing any tools or instructions, you MUST call this function to update the environment."+
			"You MUST update the environment with any useful information or tools. You will be resumed with no other context than the information provided here"),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this environment is being updated."),
		),
		mcp.WithString("environment_source",
			mcp.Description("Absolute path to the source git repository for the environment."),
			mcp.Required(),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment to update."),
			mcp.Required(),
		),
		mcp.WithString("instructions",
			mcp.Description("The instructions for the environment. This should contain any information that might be useful to operate in the environment, such as what tools are available, what commands to use to build/test/etc"),
			mcp.Required(),
		),
		mcp.WithString("title",
			mcp.Description("Short description of the work that is happening in this environment."),
			mcp.Required(),
		),
		mcp.WithString("base_image",
			mcp.Description("Change the base image for the environment."),
			mcp.Required(),
		),
		mcp.WithArray("setup_commands",
			mcp.Description("Commands that will be executed on top of the base image to set up the environment. Similar to `RUN` instructions in Dockerfiles."),
			mcp.Required(),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithArray("envs",
			mcp.Description("The environment variables to set (e.g. `[\"FOO=bar\", \"BAZ=qux\"]`)."),
			mcp.Required(),
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
			mcp.Required(),
			mcp.Items(map[string]any{"type": "string"}),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		repo, env, err := openEnvironment(ctx, request)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("unable to open the environment", err), nil
		}

		config := env.Config.Copy()

		instructions, err := request.RequireString("instructions")
		if err != nil {
			return nil, err
		}
		config.Instructions = instructions

		baseImage, err := request.RequireString("base_image")
		if err != nil {
			return nil, err
		}
		config.BaseImage = baseImage

		setupCommands, err := request.RequireStringSlice("setup_commands")
		if err != nil {
			return nil, err
		}
		config.SetupCommands = setupCommands

		envs, err := request.RequireStringSlice("envs")
		if err != nil {
			return nil, err
		}
		config.Env = envs

		secrets, err := request.RequireStringSlice("secrets")
		if err != nil {
			return nil, err
		}
		config.Secrets = secrets

		if title := request.GetString("title", ""); title != "" {
			env.State.Title = title
		}

		if err := env.UpdateConfig(ctx, request.GetString("explanation", ""), config); err != nil {
			return mcp.NewToolResultErrorFromErr("unable to update the environment", err), nil
		}

		if err := repo.Update(ctx, env, request.GetString("explanation", "")); err != nil {
			return mcp.NewToolResultErrorFromErr("unable to update the environment", err), nil
		}

		out, err := marshalEnvironment(env)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to marshal environment", err), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Environment %s updated successfully. Environment has been restarted, all previous commands have been lost.\n%s", env.ID, out)), nil
	},
}

var EnvironmentListTool = &Tool[BaseRepositoryRequest, []*EnvironmentResponse]{
	Definition: newRepositoryTool(
		"environment_list",
		"List available environments",
	),
	ParseRequest: func(request mcp.CallToolRequest) (*BaseRepositoryRequest, error) {
		base, err := parseBaseRepositoryRequest(request)
		if err != nil {
			return nil, err
		}
		return &base, nil
	},
	ActualHandler: func(ctx context.Context, request *BaseRepositoryRequest) (*ToolResponse[[]*EnvironmentResponse], error) {
		repo, err := openRepositoryFromRequest(ctx, *request)
		if err != nil {
			return nil, err
		}
		envInfos, err := repo.List(ctx)
		if err != nil {
			return nil, err
		}

		// Convert EnvironmentInfo slice to EnvironmentResponse slice
		resp := &ToolResponse[[]*EnvironmentResponse]{
			Data: make([]*EnvironmentResponse, len(envInfos)),
		}
		for i, envInfo := range envInfos {
			resp.Data[i] = environmentResponseFromEnvInfo(envInfo)
		}

		return resp, nil
	},
}

type EnvironmentRunCmdToolRequest struct {
	BaseEnvironmentRequest

	Command       string `json:"command"`
	Shell         string `json:"shell"`
	Background    bool   `json:"background"`
	UseEntrypoint bool   `json:"use_entrypoint"`
	Ports         []int  `json:"ports"`
}

var EnvironmentRunCmdTool = &Tool[EnvironmentRunCmdToolRequest, environment.EndpointMappings]{
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
	ParseRequest: func(request mcp.CallToolRequest) (*EnvironmentRunCmdToolRequest, error) {
		base, err := parseBaseEnvironmentRequest(request)
		if err != nil {
			return nil, err
		}

		req := &EnvironmentRunCmdToolRequest{
			BaseEnvironmentRequest: base,
			Command:                request.GetString("command", ""),
			Shell:                  request.GetString("shell", "sh"),
			Background:             request.GetBool("background", false),
			UseEntrypoint:          request.GetBool("use_entrypoint", false),
		}

		if portList, ok := request.GetArguments()["ports"].([]any); ok {
			for _, port := range portList {
				req.Ports = append(req.Ports, int(port.(float64)))
			}
		}

		return req, nil
	},
	ActualHandler: func(ctx context.Context, request *EnvironmentRunCmdToolRequest) (*ToolResponse[environment.EndpointMappings], error) {
		repo, env, err := openEnvironmentFromRequest(ctx, request.BaseEnvironmentRequest)
		if err != nil {
			return nil, fmt.Errorf("unable to open the environment: %w", err)
		}

		updateRepo := func() error {
			if err := repo.Update(ctx, env, request.Explanation); err != nil {
				return fmt.Errorf("failed to update repository: %w", err)
			}
			return nil
		}

		if request.Background {
			endpoints, runErr := env.RunBackground(ctx, request.Command, request.Shell, request.Ports, request.UseEntrypoint)
			// We want to update the repository even if the command failed.
			if err := updateRepo(); err != nil {
				return nil, err
			}
			if runErr != nil {
				return nil, fmt.Errorf("failed to run command: %w", runErr)
			}

			return &ToolResponse[environment.EndpointMappings]{
				Data: endpoints,
				Message: fmt.Sprintf(`Command started in the background in NEW container.

To access from the user's machine: use host_external. To access from other commands in this environment: use environment_internal.

Any changes to the container workdir (%s) WILL NOT be committed to container-use/%s

Background commands are unaffected by filesystem and any other kind of changes. You need to start a new command for changes to take effect.`,
					env.Config.Workdir, env.ID),
			}, nil
		}

		stdout, runErr := env.Run(ctx, request.Command, request.Shell, request.UseEntrypoint)
		// We want to update the repository even if the command failed.
		if err := updateRepo(); err != nil {
			return nil, err
		}
		if runErr != nil {
			return nil, fmt.Errorf("failed to run command: %w", runErr)
		}

		return &ToolResponse[environment.EndpointMappings]{
			Message: fmt.Sprintf("%s\n\nAny changes to the container workdir (%s) have been committed and pushed to container-use/ remote", stdout, env.Config.Workdir),
		}, nil

	},
}

var EnvironmentFileReadTool = &Tool[Request, Response]{
	Definition: mcp.NewTool("environment_file_read",
		mcp.WithDescription("Read the contents of a file, specifying a line range or the entire file."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this file is being read."),
		),
		mcp.WithString("environment_source",
			mcp.Description("Absolute path to the source git repository for the environment."),
			mcp.Required(),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment for this command. Must call `environment_create` first."),
			mcp.Required(),
		),
		mcp.WithString("target_file",
			mcp.Description("Path of the file to read, absolute or relative to the workdir"),
			mcp.Required(),
		),
		mcp.WithBoolean("should_read_entire_file",
			mcp.Description("Whether to read the entire file. Defaults to false."),
		),
		mcp.WithNumber("start_line_one_indexed",
			mcp.Description("The one-indexed line number to start reading from (inclusive)."),
		),
		mcp.WithNumber("end_line_one_indexed_inclusive",
			mcp.Description("The one-indexed line number to end reading at (inclusive)."),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_, env, err := openEnvironment(ctx, request)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("unable to open the environment", err), nil
		}

		targetFile, err := request.RequireString("target_file")
		if err != nil {
			return nil, err
		}
		shouldReadEntireFile := request.GetBool("should_read_entire_file", false)
		startLineOneIndexed := request.GetInt("start_line_one_indexed", 0)
		endLineOneIndexedInclusive := request.GetInt("end_line_one_indexed_inclusive", 0)

		fileContents, err := env.FileRead(ctx, targetFile, shouldReadEntireFile, startLineOneIndexed, endLineOneIndexedInclusive)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to read file", err), nil
		}

		return mcp.NewToolResultText(fileContents), nil
	},
}

type EnvironmentFileListToolRequest struct {
	BaseEnvironmentRequest

	Path string `json:"path"`
}

var EnvironmentFileListTool = &Tool[EnvironmentFileListToolRequest, string]{
	Definition: newEnvironmentTool(
		"environment_file_list",
		"List the contents of a directory",
		mcp.WithString("path",
			mcp.Description("Path of the directory to list contents of, absolute or relative to the workdir"),
			mcp.Required(),
		),
	),
	ParseRequest: func(request mcp.CallToolRequest) (*EnvironmentFileListToolRequest, error) {
		base, err := parseBaseEnvironmentRequest(request)
		if err != nil {
			return nil, err
		}

		path, err := request.RequireString("path")
		if err != nil {
			return nil, err
		}

		return &EnvironmentFileListToolRequest{
			BaseEnvironmentRequest: base,
			Path:                   path,
		}, nil
	},
	ActualHandler: func(ctx context.Context, request *EnvironmentFileListToolRequest) (*ToolResponse[string], error) {
		_, env, err := openEnvironmentFromRequest(ctx, request.BaseEnvironmentRequest)
		if err != nil {
			return nil, fmt.Errorf("unable to open the environment: %w", err)
		}

		out, err := env.FileList(ctx, request.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to list directory: %w", err)
		}

		return &ToolResponse[string]{
			Data: out,
		}, nil
	},
}

var EnvironmentFileWriteTool = &Tool[Request, Response]{
	Definition: mcp.NewTool("environment_file_write",
		mcp.WithDescription("Write the contents of a file."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this file is being written."),
		),
		mcp.WithString("environment_source",
			mcp.Description("Absolute path to the source git repository for the environment."),
			mcp.Required(),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment for this command. Must call `environment_create` first."),
			mcp.Required(),
		),
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
			return mcp.NewToolResultErrorFromErr("unable to open the environment", err), nil
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
			return mcp.NewToolResultErrorFromErr("failed to write file", err), nil
		}

		if err := repo.Update(ctx, env, request.GetString("explanation", "")); err != nil {
			return mcp.NewToolResultErrorFromErr("unable to update the environment", err), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("file %s written successfully and committed to container-use/ remote", targetFile)), nil
	},
}

var EnvironmentFileDeleteTool = &Tool[Request, Response]{
	Definition: mcp.NewTool("environment_file_delete",
		mcp.WithDescription("Deletes a file at the specified path."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this file is being deleted."),
		),
		mcp.WithString("environment_source",
			mcp.Description("Absolute path to the source git repository for the environment."),
			mcp.Required(),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment for this command. Must call `environment_create` first."),
			mcp.Required(),
		),
		mcp.WithString("target_file",
			mcp.Description("Path of the file to delete, absolute or relative to the workdir."),
			mcp.Required(),
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

		if err := env.FileDelete(ctx, request.GetString("explanation", ""), targetFile); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to delete file", err), nil
		}

		if err := repo.Update(ctx, env, request.GetString("explanation", "")); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to update env", err), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("file %s deleted successfully and committed to container-use/ remote", targetFile)), nil
	},
}

var EnvironmentCheckpointTool = &Tool[Request, Response]{
	Definition: mcp.NewTool("environment_checkpoint",
		mcp.WithDescription("Checkpoints an environment in its current state as a container."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this checkpoint is being created."),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment for this command. Must call `environment_create` first."),
			mcp.Required(),
		),
		mcp.WithString("destination",
			mcp.Description("Container image destination to checkpoint to (e.g. registry.com/user/image:tag"),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_, env, err := openEnvironment(ctx, request)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("unable to open the environment", err), nil
		}
		destination, err := request.RequireString("destination")
		if err != nil {
			return nil, err
		}

		endpoint, err := env.Checkpoint(ctx, destination)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to checkpoint", err), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Checkpoint pushed to %q. You MUST use the full content addressed (@sha256:...) reference in `docker` commands. The entrypoint is set to `sh`, keep that in mind when giving commands to the container.", endpoint)), nil
	},
}

var EnvironmentAddServiceTool = &Tool[Request, Response]{
	Definition: mcp.NewTool("environment_add_service",
		mcp.WithDescription("Add a service to the environment (e.g. database, cache, etc.)"),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this service is being added."),
		),
		mcp.WithString("environment_source",
			mcp.Description("Absolute path to the source git repository for the environment."),
			mcp.Required(),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment for this command. Must call `environment_create` first."),
			mcp.Required(),
		),
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
			return mcp.NewToolResultErrorFromErr("unable to open the environment", err), nil
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
			return mcp.NewToolResultErrorFromErr("failed to add service", err), nil
		}

		if err := repo.Update(ctx, env, request.GetString("explanation", "")); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to update env", err), nil
		}

		output, err := json.Marshal(service)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to marshal service", err), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Service added and started successfully: %s", string(output))), nil
	},
}
