package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Tool struct {
	Definition mcp.Tool
	Handler    server.ToolHandlerFunc
}

var tools = []*Tool{}

func RegisterTool(tool ...*Tool) {
	tools = append(tools, tool...)
}

func init() {
	RegisterTool(
		EnvironmentOpenTool,
		EnvironmentUpdateTool,
		// EnvironmentCreateTool,
		// EnvironmentListTool,
		// EnvironmentHistoryTool,
		// EnvironmentRevertTool,
		// EnvironmentForkTool,

		EnvironmentRunCmdTool,
		// EnvironmentSetEnvTool,

		// EnvironmentUploadTool,
		// EnvironmentDownloadTool,
		// EnvironmentDiffTool,

		EnvironmentFileReadTool,
		EnvironmentFileListTool,
		EnvironmentFileWriteTool,
		EnvironmentFileDeleteTool,
		// EnvironmentRevisionDiffTool,
	)
}

var EnvironmentOpenTool = &Tool{
	Definition: mcp.NewTool("environment_open",
		mcp.WithDescription(`Opens (or creates) a development environment. The environment is the result of a build of the Dockerfile specification. Read carefully the instructions to understand the environment. DO NOT manually install toolchains inside the environment, instead explicitly call environment_update"`),
		mcp.WithString("source",
			mcp.Description("The source directory of the environment."), //  This can be a local folder (e.g. file://) or a URL to a git repository (e.g. https://github.com/user/repo.git, git@github.com:user/repo.git)"),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		source, err := request.RequireString("source")
		if err != nil {
			return nil, err
		}
		env, err := OpenEnvironment(ctx, source)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to open environment", err), nil
		}
		out, err := json.Marshal(env)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	},
}

var EnvironmentUpdateTool = &Tool{
	Definition: mcp.NewTool("environment_update",
		mcp.WithDescription(`Updates an environment with new instructions and toolchains. If the environment is missing any tools or instructions, you MUST call this function to update the environment.`),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this environment is being updated."),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment to update."),
			mcp.Required(),
		),
		mcp.WithString("instructions",
			mcp.Description("The instructions for the environment. This should contain any information that might be useful to operate in the environment, such as what tools are available, what commands to use to build/test/etc"),
			mcp.Required(),
		),
		mcp.WithString("dockerfile",
			mcp.Description("Dockerfile used to build the environment. Please update this with anything that might be useful to operate in the environment"),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		environmentID, err := request.RequireString("environment_id")
		if err != nil {
			return nil, err
		}
		environment := GetEnvironment(environmentID)
		if environment == nil {
			return nil, errors.New("environment not found")
		}
		instructions, err := request.RequireString("instructions")
		if err != nil {
			return nil, err
		}
		dockerfile, err := request.RequireString("dockerfile")
		if err != nil {
			return nil, err
		}
		if err := environment.Update(ctx, request.GetString("explanation", ""), dockerfile, instructions); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to update environment", err), nil
		}
		out, err := json.Marshal(environment)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
		// source, err := request.RequireString("source")
		// if err != nil {
		// 	return nil, err
		// }
		// return mcp.NewToolResultText(
		// 	fmt.Sprintf(`Environment for %q not found, created a brand new one (ID: %q) using "ubuntu:latest" as base image. If you need tools or further instructions, please look around the filesystem and update the environment.`, source),
		// ), nil

		// // env, err := OpenEnvironment(source)
		// // if err != nil {
		// // 	return mcp.NewToolResultErrorFromErr("failed to open environment", err), nil
		// // }
		// return mcp.NewToolResultText(fmt.Sprintf(`{"id": %q}`, sandbox.ID)), nil
	},
}

var EnvironmentListTool = &Tool{
	Definition: mcp.NewTool("environment_list",
		mcp.WithDescription("List available environments"),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this environment is being listed."),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		environments := ListEnvironments()
		out, err := json.Marshal(environments)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	},
}

var EnvironmentForkTool = &Tool{
	Definition: mcp.NewTool("environment_fork",
		mcp.WithDescription("Create a new environment from an existing environment."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this environment is being forked."),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment to fork."),
			mcp.Required(),
		),
		mcp.WithNumber("version",
			mcp.Description("Version of the environment to fork. Defaults to latest version."),
		),
		mcp.WithString("name",
			mcp.Description("Name of the new environment."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		environmentID, err := request.RequireString("environment_id")
		if err != nil {
			return nil, err
		}

		environment := GetEnvironment(environmentID)
		if environment == nil {
			return nil, errors.New("environment not found")
		}

		name, err := request.RequireString("name")
		if err != nil {
			return nil, err
		}

		var version *Version
		if v, ok := request.GetArguments()["version"].(Version); ok {
			version = &v
		}

		fork, err := environment.Fork(ctx, request.GetString("explanation", ""), name, version)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to fork environment", err), nil
		}

		return mcp.NewToolResultText("environment forked successfully into ID " + fork.ID), nil
	},
}

var EnvironmentHistoryTool = &Tool{
	Definition: mcp.NewTool("environment_history",
		mcp.WithDescription("List the history of an environment."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this environment is being listed."),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment for this command. Must call `environment_create` first."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		environmentID, err := request.RequireString("environment_id")
		if err != nil {
			return nil, err
		}

		environment := GetEnvironment(environmentID)
		if environment == nil {
			return nil, errors.New("environment not found")
		}

		history := environment.History
		out, err := json.Marshal(history)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	},
}

var EnvironmentRevertTool = &Tool{
	Definition: mcp.NewTool("environment_revert",
		mcp.WithDescription("Revert the environment to a specific version."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this environment is being listed."),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment for this command. Must call `environment_create` first."),
			mcp.Required(),
		),
		mcp.WithNumber("version",
			mcp.Description("The version to revert to."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		environmentID, err := request.RequireString("environment_id")
		if err != nil {
			return nil, err
		}

		environment := GetEnvironment(environmentID)
		if environment == nil {
			return nil, errors.New("environment not found")
		}

		version, err := request.RequireInt("version")
		if err != nil {
			return nil, err
		}

		if err := environment.Revert(ctx, request.GetString("explanation", ""), Version(version)); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to revert environment", err), nil
		}

		return mcp.NewToolResultText("environment reverted successfully"), nil
	},
}

var EnvironmentRunCmdTool = &Tool{
	Definition: mcp.NewTool("environment_run_cmd",
		mcp.WithDescription("Run a command on behalf of the user in the terminal."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this command is being run."),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment for this command. Must call `environment_create` first."),
			mcp.Required(),
		),
		mcp.WithString("command",
			mcp.Description("The terminal command to execute. If empty, the environment's default command is used."),
		),
		mcp.WithString("shell",
			mcp.Description("The shell that will be interpreting this command (default: sh)"),
		),
		mcp.WithBoolean("background",
			mcp.Description("Run the command in the background. Must ALWAYS be set for long running command (e.g. http server). Failure to do so will result in the tool being stuck, awaiting for the command to finish."),
		),
		mcp.WithBoolean("use_entrypoint",
			mcp.Description("Use the image entrypoint, if present, by prepending it to the args."),
		),
		mcp.WithArray("ports",
			mcp.Description("Ports to expose. Only works with background environments. For each port, returns the internal (for use by other environments) and external (for use by the user) address."),
			mcp.Items(map[string]any{"type": "number"}),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		environmentID, err := request.RequireString("environment_id")
		if err != nil {
			return nil, err
		}
		environment := GetEnvironment(environmentID)
		if environment == nil {
			return nil, errors.New("environment not found")
		}
		command := request.GetString("command", "")
		shell := request.GetString("shell", "sh")

		background := request.GetBool("background", false)
		if background {
			ports := []int{}
			if portList, ok := request.GetArguments()["ports"].([]any); ok {
				for _, port := range portList {
					ports = append(ports, int(port.(float64)))
				}
			}
			endpoints, err := environment.RunBackground(ctx, request.GetString("explanation", ""), command, shell, ports, request.GetBool("use_entrypoint", false))
			if err != nil {
				return mcp.NewToolResultErrorFromErr("failed to run command", err), nil
			}

			out, err := json.Marshal(endpoints)
			if err != nil {
				return nil, err
			}

			return mcp.NewToolResultText(fmt.Sprintf("Command started in the background. Endpoints are %s", string(out))), nil
		}

		stdout, err := environment.Run(ctx, request.GetString("explanation", ""), command, shell, request.GetBool("use_entrypoint", false))
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to run command", err), nil
		}
		return mcp.NewToolResultText(stdout), nil
	},
}

var EnvironmentSetEnvTool = &Tool{
	Definition: mcp.NewTool("environment_set_env",
		mcp.WithDescription("Set environment variables for an environment."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why these environment variables are being set."),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment for this command. Must call `environment_create` first."),
			mcp.Required(),
		),
		mcp.WithArray("envs",
			mcp.Description("The environment variables to set."),
			mcp.Items(map[string]any{"type": "string"}),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		environmentID, err := request.RequireString("environment_id")
		if err != nil {
			return nil, err
		}
		environment := GetEnvironment(environmentID)
		if environment == nil {
			return nil, errors.New("environment not found")
		}
		envs, err := request.RequireStringSlice("envs")
		if err != nil {
			return nil, err
		}
		if err := environment.SetEnv(ctx, request.GetString("explanation", ""), envs); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to set environment variables", err), nil
		}
		return mcp.NewToolResultText("environment variables set successfully"), nil
	},
}

var EnvironmentUploadTool = &Tool{
	Definition: mcp.NewTool("environment_upload",
		mcp.WithDescription("Upload files to an environment."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this file is being uploaded."),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment for this command. Must call `environment_create` first."),
			mcp.Required(),
		),
		mcp.WithString("source",
			mcp.Description("The source directory to be uploaded to the environment. This can be a local folder (e.g. file://) or a URL to a git repository (e.g. https://github.com/user/repo.git, git@github.com:user/repo.git)"),
			mcp.Required(),
		),
		mcp.WithString("target",
			mcp.Description("The target destination in the environment where to upload files."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		environmentID, err := request.RequireString("environment_id")
		if err != nil {
			return nil, err
		}
		environment := GetEnvironment(environmentID)
		if environment == nil {
			return nil, errors.New("environment not found")
		}

		source, err := request.RequireString("source")
		if err != nil {
			return nil, err
		}
		target, err := request.RequireString("target")
		if err != nil {
			return nil, err
		}

		if err := environment.Upload(ctx, request.GetString("explanation", ""), source, target); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to upload files", err), nil
		}

		return mcp.NewToolResultText("files uploaded successfully"), nil
	},
}

var EnvironmentDownloadTool = &Tool{
	Definition: mcp.NewTool("environment_download",
		mcp.WithDescription("Download files from an environment to the local filesystem."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this file is being downloaded."),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment for this command. Must call `environment_create` first."),
			mcp.Required(),
		),
		mcp.WithString("source",
			mcp.Description("The source directory to be downloaded from the environment."),
			mcp.Required(),
		),
		mcp.WithString("target",
			mcp.Description("The target destination on the local filesystem where to download files."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		environmentID, err := request.RequireString("environment_id")
		if err != nil {
			return nil, err
		}
		environment := GetEnvironment(environmentID)
		if environment == nil {
			return nil, errors.New("environment not found")
		}

		source, err := request.RequireString("source")
		if err != nil {
			return nil, err
		}
		target, err := request.RequireString("target")
		if err != nil {
			return nil, errors.New("target must be a string")
		}

		if err := environment.Download(ctx, source, target); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to download files", err), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("files downloaded successfully to %s", target)), nil
	},
}

var EnvironmentDiffTool = &Tool{
	Definition: mcp.NewTool("environment_remote_diff",
		mcp.WithDescription("Diff files between an environment and the local filesystem or git repository."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this diff is being run."),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment for this command. Must call `environment_create` first."),
			mcp.Required(),
		),
		mcp.WithString("source",
			mcp.Description("The source directory to be compared. This can be a local folder (e.g. file://) or a URL to a git repository (e.g. https://github.com/user/repo.git, git@github.com:user/repo.git)"),
			mcp.Required(),
		),
		mcp.WithString("target",
			mcp.Description("The target destination on the environment filesystem where to compare against."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		environmentID, err := request.RequireString("environment_id")
		if err != nil {
			return nil, err
		}
		environment := GetEnvironment(environmentID)
		if environment == nil {
			return nil, errors.New("environment not found")
		}

		source, err := request.RequireString("source")
		if err != nil {
			return nil, err
		}
		target, err := request.RequireString("target")
		if err != nil {
			return nil, errors.New("target must be a string")
		}

		diff, err := environment.RemoteDiff(ctx, source, target)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to diff", err), nil
		}

		return mcp.NewToolResultText(diff), nil
	},
}

var EnvironmentFileReadTool = &Tool{
	Definition: mcp.NewTool("environment_file_read",
		mcp.WithDescription("Read the contents of a file, specifying a line range or the entire file."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this file is being read."),
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
		environmentID, err := request.RequireString("environment_id")
		if err != nil {
			return nil, err
		}
		environment := GetEnvironment(environmentID)
		if environment == nil {
			return nil, errors.New("environment not found")
		}

		targetFile, err := request.RequireString("target_file")
		if err != nil {
			return nil, err
		}
		shouldReadEntireFile := request.GetBool("should_read_entire_file", false)
		startLineOneIndexed := request.GetInt("start_line_one_indexed", 0)
		endLineOneIndexedInclusive := request.GetInt("end_line_one_indexed_inclusive", 0)

		fileContents, err := environment.FileRead(ctx, targetFile, shouldReadEntireFile, startLineOneIndexed, endLineOneIndexedInclusive)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to read file", err), nil
		}

		return mcp.NewToolResultText(fileContents), nil
	},
}

var EnvironmentFileListTool = &Tool{
	Definition: mcp.NewTool("environment_file_list",
		mcp.WithDescription("List the contents of a directory"),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this directory is being listed."),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment for this command. Must call `environment_create` first."),
			mcp.Required(),
		),
		mcp.WithString("path",
			mcp.Description("Path of the directory to list contents of, absolute or relative to the workdir"),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		environmentID, err := request.RequireString("environment_id")
		if err != nil {
			return nil, err
		}
		environment := GetEnvironment(environmentID)
		if environment == nil {
			return nil, errors.New("environment not found")
		}

		path, err := request.RequireString("path")
		if err != nil {
			return nil, err
		}

		out, err := environment.FileList(ctx, path)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to list directory", err), nil
		}

		return mcp.NewToolResultText(out), nil
	},
}

var EnvironmentFileWriteTool = &Tool{
	Definition: mcp.NewTool("environment_file_write",
		mcp.WithDescription("Write the contents of a file."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this file is being written."),
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
		environmentID, err := request.RequireString("environment_id")
		if err != nil {
			return nil, err
		}
		environment := GetEnvironment(environmentID)
		if environment == nil {
			return nil, errors.New("environment not found")
		}

		targetFile, err := request.RequireString("target_file")
		if err != nil {
			return nil, err
		}
		contents, err := request.RequireString("contents")
		if err != nil {
			return nil, err
		}

		if err := environment.FileWrite(ctx, request.GetString("explanation", ""), targetFile, contents); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to write file", err), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("file %s written successfully", targetFile)), nil
	},
}

var EnvironmentFileDeleteTool = &Tool{
	Definition: mcp.NewTool("environment_file_delete",
		mcp.WithDescription("Deletes a file at the specified path."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this file is being deleted."),
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
		environmentID, err := request.RequireString("environment_id")
		if err != nil {
			return nil, err
		}
		environment := GetEnvironment(environmentID)
		if environment == nil {
			return nil, errors.New("environment not found")
		}

		targetFile, err := request.RequireString("target_file")
		if err != nil {
			return nil, err
		}

		if err := environment.FileDelete(ctx, request.GetString("explanation", ""), targetFile); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to delete file", err), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("file %s deleted successfully", targetFile)), nil
	},
}

var EnvironmentRevisionDiffTool = &Tool{
	Definition: mcp.NewTool("environment_revision_diff",
		mcp.WithDescription("Diff files between multiple revisions of an environment."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this diff is being run."),
		),
		mcp.WithString("environment_id",
			mcp.Description("The ID of the environment for this command. Must call `environment_create` first."),
			mcp.Required(),
		),
		mcp.WithString("path",
			mcp.Description("The path within the environment to be diffed. Defaults to workdir."),
		),
		mcp.WithNumber("from_version",
			mcp.Description("Compute the diff starting from this version"),
			mcp.Required(),
		),
		mcp.WithNumber("to_version",
			mcp.Description("Compute the diff ending at this version. Defaults to latest version."),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		environmentID, err := request.RequireString("environment_id")
		if err != nil {
			return nil, err
		}
		environment := GetEnvironment(environmentID)
		if environment == nil {
			return nil, errors.New("environment not found")
		}

		path := request.GetString("path", "")
		fromVersion, err := request.RequireInt("from_version")
		if err != nil {
			return nil, err
		}
		toVersion := request.GetInt("to_version", int(environment.History.LatestVersion()))

		diff, err := environment.RevisionDiff(ctx, path, Version(fromVersion), Version(toVersion))
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to diff", err), nil
		}

		return mcp.NewToolResultText(diff), nil
	},
}
