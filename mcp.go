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
		ContainerCreateTool,
		ContainerListTool,
		ContainerHistoryTool,
		ContainerRevertTool,
		ContainerForkTool,

		ContainerRunCmdTool,

		ContainerUploadTool,
		ContainerDownloadTool,
		ContainerDiffTool,

		ContainerFileReadTool,
		ContainerFileListTool,
		ContainerFileWriteTool,
		ContainerFileDeleteTool,

		HostDirectoryListTool,
		HostDirectoryHistoryTool,
		HostDirectoryRevertTool,
	)
}

var ContainerCreateTool = &Tool{
	Definition: mcp.NewTool("container_create",
		mcp.WithDescription(`Create a new container. The sandbox only contains the base image specified, anything else required will need to be installed by hand.
		You won't be able to access your local filesystem directly. If you need to manipulate the filesystem, first you will have to call "container_upload"`),
		mcp.WithString("name",
			mcp.Description("The name of the container."),
			mcp.Required(),
		),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this sandbox is being created."),
		),
		mcp.WithString("image",
			mcp.Description("The base image this workspace will use (e.g. alpine:latest, ubuntu:24.04, etc.)"),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return nil, err
		}
		image, err := request.RequireString("image")
		if err != nil {
			return nil, err
		}
		sandbox, err := CreateContainer(name, request.GetString("explanation", ""), image)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to create container", err), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf(`{"id": %q}`, sandbox.ID)), nil
	},
}

var ContainerListTool = &Tool{
	Definition: mcp.NewTool("container_list",
		mcp.WithDescription("List available containers"),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this container is being listed."),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		containers := ListContainers()
		out, err := json.Marshal(containers)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	},
}

var ContainerForkTool = &Tool{
	Definition: mcp.NewTool("container_fork",
		mcp.WithDescription("Create a new container from an existing container."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this container is being forked."),
		),
		mcp.WithString("container_id",
			mcp.Description("The ID of the container to fork."),
			mcp.Required(),
		),
		mcp.WithNumber("version",
			mcp.Description("Version of the container to fork. Defaults to latest version."),
		),
		mcp.WithString("name",
			mcp.Description("Name of the new container."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		containerID, err := request.RequireString("container_id")
		if err != nil {
			return nil, err
		}

		container := GetContainer(containerID)
		if container == nil {
			return nil, errors.New("container not found")
		}

		name, err := request.RequireString("name")
		if err != nil {
			return nil, err
		}

		var version *Version
		if v, ok := request.GetArguments()["version"].(Version); ok {
			version = &v
		}

		fork, err := container.Fork(ctx, request.GetString("explanation", ""), name, version)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to fork container", err), nil
		}

		return mcp.NewToolResultText("container forked successfully into ID " + fork.ID), nil
	},
}

var ContainerHistoryTool = &Tool{
	Definition: mcp.NewTool("container_history",
		mcp.WithDescription("List the history of a container."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this container is being listed."),
		),
		mcp.WithString("container_id",
			mcp.Description("The ID of the container for this command. Must call `container_create` first."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		containerID, err := request.RequireString("container_id")
		if err != nil {
			return nil, err
		}

		container := GetContainer(containerID)
		if container == nil {
			return nil, errors.New("container not found")
		}

		history := container.History
		out, err := json.Marshal(history)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	},
}

var ContainerRevertTool = &Tool{
	Definition: mcp.NewTool("container_revert",
		mcp.WithDescription("Revert the container to a specific version."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this container is being listed."),
		),
		mcp.WithString("container_id",
			mcp.Description("The ID of the container for this command. Must call `container_create` first."),
			mcp.Required(),
		),
		mcp.WithNumber("version",
			mcp.Description("The version to revert to."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		containerID, err := request.RequireString("container_id")
		if err != nil {
			return nil, err
		}

		container := GetContainer(containerID)
		if container == nil {
			return nil, errors.New("container not found")
		}

		version, err := request.RequireInt("version")
		if err != nil {
			return nil, err
		}

		if err := container.Revert(ctx, request.GetString("explanation", ""), Version(version)); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to revert container", err), nil
		}

		return mcp.NewToolResultText("container reverted successfully"), nil
	},
}

var ContainerRunCmdTool = &Tool{
	Definition: mcp.NewTool("container_run_cmd",
		mcp.WithDescription("Run a command on behalf of the user in the terminal."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this command is being run."),
		),
		mcp.WithString("container_id",
			mcp.Description("The ID of the container for this command. Must call `container_create` first."),
			mcp.Required(),
		),
		mcp.WithString("command",
			mcp.Description("The terminal command to execute"),
			mcp.Required(),
		),
		mcp.WithString("shell",
			mcp.Description("The shell that will be interpreting this command (default: sh)"),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		containerID, err := request.RequireString("container_id")
		if err != nil {
			return nil, err
		}
		container := GetContainer(containerID)
		if container == nil {
			return nil, errors.New("container not found")
		}
		command, err := request.RequireString("command")
		if err != nil {
			return nil, errors.New("command must be a string")
		}
		shell, ok := request.GetArguments()["shell"].(string)
		if !ok {
			shell = "bash"
		}
		stdout, err := container.Run(ctx, request.GetString("explanation", ""), command, shell)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to run command", err), nil
		}
		return mcp.NewToolResultText(stdout), nil
	},
}

var ContainerUploadTool = &Tool{
	Definition: mcp.NewTool("container_upload",
		mcp.WithDescription("Upload files to a container."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this file is being uploaded."),
		),
		mcp.WithString("container_id",
			mcp.Description("The ID of the container for this command. Must call `container_create` first."),
			mcp.Required(),
		),
		mcp.WithString("source",
			mcp.Description("The source directory to be uploaded to the container. This can be a local folder (e.g. file://) or a URL to a git repository (e.g. https://github.com/user/repo.git, git@github.com:user/repo.git)"),
			mcp.Required(),
		),
		mcp.WithString("target",
			mcp.Description("The target destination in the container where to upload files."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		containerID, err := request.RequireString("container_id")
		if err != nil {
			return nil, err
		}
		container := GetContainer(containerID)
		if container == nil {
			return nil, errors.New("container not found")
		}

		source, err := request.RequireString("source")
		if err != nil {
			return nil, err
		}
		target, err := request.RequireString("target")
		if err != nil {
			return nil, err
		}

		if err := container.Upload(ctx, request.GetString("explanation", ""), source, target); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to upload files", err), nil
		}

		return mcp.NewToolResultText("files uploaded successfully"), nil
	},
}

var ContainerDownloadTool = &Tool{
	Definition: mcp.NewTool("container_download",
		mcp.WithDescription("Download files from a container to the local filesystem."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this file is being downloaded."),
		),
		mcp.WithString("container_id",
			mcp.Description("The ID of the container for this command. Must call `container_create` first."),
			mcp.Required(),
		),
		mcp.WithString("source",
			mcp.Description("The source directory to be downloaded from the container."),
			mcp.Required(),
		),
		mcp.WithString("target",
			mcp.Description("The target destination on the local filesystem where to download files."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		containerID, err := request.RequireString("container_id")
		if err != nil {
			return nil, err
		}
		container := GetContainer(containerID)
		if container == nil {
			return nil, errors.New("container not found")
		}

		source, err := request.RequireString("source")
		if err != nil {
			return nil, err
		}
		target, err := request.RequireString("target")
		if err != nil {
			return nil, errors.New("target must be a string")
		}

		if err := container.Download(ctx, source, target); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to download files", err), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("files downloaded successfully to %s", target)), nil
	},
}

var ContainerDiffTool = &Tool{
	Definition: mcp.NewTool("container_diff",
		mcp.WithDescription("Diff files between a container and the local filesystem."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this diff is being run."),
		),
		mcp.WithString("container_id",
			mcp.Description("The ID of the container for this command. Must call `container_create` first."),
			mcp.Required(),
		),
		mcp.WithString("source",
			mcp.Description("The source directory to be compared. This can be a local folder (e.g. file://) or a URL to a git repository (e.g. https://github.com/user/repo.git, git@github.com:user/repo.git)"),
			mcp.Required(),
		),
		mcp.WithString("target",
			mcp.Description("The target destination on the container filesystem where to compare against."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		containerID, err := request.RequireString("container_id")
		if err != nil {
			return nil, err
		}
		container := GetContainer(containerID)
		if container == nil {
			return nil, errors.New("container not found")
		}

		source, err := request.RequireString("source")
		if err != nil {
			return nil, err
		}
		target, err := request.RequireString("target")
		if err != nil {
			return nil, errors.New("target must be a string")
		}

		diff, err := container.Diff(ctx, source, target)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to diff", err), nil
		}

		return mcp.NewToolResultText(diff), nil
	},
}

var ContainerFileReadTool = &Tool{
	Definition: mcp.NewTool("container_file_read",
		mcp.WithDescription("Read the contents of a file, specifying a line range or the entire file."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this file is being read."),
		),
		mcp.WithString("container_id",
			mcp.Description("The ID of the container for this command. Must call `container_create` first."),
			mcp.Required(),
		),
		mcp.WithString("target_file",
			mcp.Description("Path of the file to read."),
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
		containerID, err := request.RequireString("container_id")
		if err != nil {
			return nil, err
		}
		container := GetContainer(containerID)
		if container == nil {
			return nil, errors.New("container not found")
		}

		targetFile, err := request.RequireString("target_file")
		if err != nil {
			return nil, err
		}
		shouldReadEntireFile := request.GetBool("should_read_entire_file", false)
		startLineOneIndexed := request.GetInt("start_line_one_indexed", 0)
		endLineOneIndexedInclusive := request.GetInt("end_line_one_indexed_inclusive", 0)

		fileContents, err := container.FileRead(ctx, targetFile, shouldReadEntireFile, startLineOneIndexed, endLineOneIndexedInclusive)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to read file", err), nil
		}

		return mcp.NewToolResultText(fileContents), nil
	},
}

var ContainerFileListTool = &Tool{
	Definition: mcp.NewTool("container_file_list",
		mcp.WithDescription("List the contents of a directory"),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this directory is being listed."),
		),
		mcp.WithString("container_id",
			mcp.Description("The ID of the container for this command. Must call `container_create` first."),
			mcp.Required(),
		),
		mcp.WithString("path",
			mcp.Description("Path of the directory to list contents of."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		containerID, err := request.RequireString("container_id")
		if err != nil {
			return nil, err
		}
		container := GetContainer(containerID)
		if container == nil {
			return nil, errors.New("container not found")
		}

		path, err := request.RequireString("path")
		if err != nil {
			return nil, err
		}

		out, err := container.FileList(ctx, path)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to list directory", err), nil
		}

		return mcp.NewToolResultText(out), nil
	},
}

var ContainerFileWriteTool = &Tool{
	Definition: mcp.NewTool("container_file_write",
		mcp.WithDescription("Write the contents of a file."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this file is being written."),
		),
		mcp.WithString("container_id",
			mcp.Description("The ID of the container for this command. Must call `container_create` first."),
			mcp.Required(),
		),
		mcp.WithString("target_file",
			mcp.Description("Path of the file to write."),
			mcp.Required(),
		),
		mcp.WithString("contents",
			mcp.Description("Full text content of the file you want to write."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		containerID, err := request.RequireString("container_id")
		if err != nil {
			return nil, err
		}
		container := GetContainer(containerID)
		if container == nil {
			return nil, errors.New("container not found")
		}

		targetFile, err := request.RequireString("target_file")
		if err != nil {
			return nil, err
		}
		contents, err := request.RequireString("contents")
		if err != nil {
			return nil, err
		}

		if err := container.FileWrite(ctx, request.GetString("explanation", ""), targetFile, contents); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to write file", err), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("file %s written successfully", targetFile)), nil
	},
}

var ContainerFileDeleteTool = &Tool{
	Definition: mcp.NewTool("container_file_delete",
		mcp.WithDescription("Deletes a file at the specified path."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this file is being deleted."),
		),
		mcp.WithString("container_id",
			mcp.Description("The ID of the container for this command. Must call `container_create` first."),
			mcp.Required(),
		),
		mcp.WithString("target_file",
			mcp.Description("Path of the file to delete."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		containerID, err := request.RequireString("container_id")
		if err != nil {
			return nil, err
		}
		container := GetContainer(containerID)
		if container == nil {
			return nil, errors.New("container not found")
		}

		targetFile, err := request.RequireString("target_file")
		if err != nil {
			return nil, err
		}

		if err := container.FileDelete(ctx, request.GetString("explanation", ""), targetFile); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to delete file", err), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("file %s deleted successfully", targetFile)), nil
	},
}

var HostDirectoryListTool = &Tool{
	Definition: mcp.NewTool("host_directory_list",
		mcp.WithDescription("List available host directories"),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why host directories are being listed."),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hostDirectories := ListHostDirectories()
		out, err := json.Marshal(hostDirectories)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	},
}

var HostDirectoryHistoryTool = &Tool{
	Definition: mcp.NewTool("host_directory_history",
		mcp.WithDescription("List the history of a host directory."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this host directory history is being listed."),
		),
		mcp.WithString("path",
			mcp.Description("The path of the host directory to get history for."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path, err := request.RequireString("path")
		if err != nil {
			return nil, err
		}

		hostDirectory := GetHostDirectory(path)
		history := hostDirectory.History
		out, err := json.Marshal(history)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	},
}

var HostDirectoryRevertTool = &Tool{
	Definition: mcp.NewTool("host_directory_revert",
		mcp.WithDescription("Revert the host directory to a specific version."),
		mcp.WithString("explanation",
			mcp.Description("One sentence explanation for why this host directory is being reverted."),
		),
		mcp.WithString("path",
			mcp.Description("The path of the host directory to revert."),
			mcp.Required(),
		),
		mcp.WithNumber("version",
			mcp.Description("The version to revert to."),
			mcp.Required(),
		),
	),
	Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path, err := request.RequireString("path")
		if err != nil {
			return nil, err
		}

		hostDirectory := GetHostDirectory(path)

		version, err := request.RequireInt("version")
		if err != nil {
			return nil, err
		}

		if err := hostDirectory.Revert(ctx, request.GetString("explanation", ""), Version(version)); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to revert host directory", err), nil
		}

		return mcp.NewToolResultText("host directory reverted successfully"), nil
	},
}
