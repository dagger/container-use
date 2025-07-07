package main

import (
	"log/slog"
	"os"

	"dagger.io/dagger"
	"github.com/dagger/container-use/mcpserver"
	"github.com/spf13/cobra"
)

var stdioCmd = &cobra.Command{
	Use:   "stdio",
	Short: "Start MCP server for agent integration",
	Long:  `Start the Model Context Protocol server that enables AI agents to create and manage containerized environments. This is typically used by agents like Claude Code, Cursor, or VSCode.`,
	RunE: func(app *cobra.Command, _ []string) error {
		ctx := app.Context()

		slog.Info("connecting to dagger")

		dag, err := dagger.Connect(ctx, dagger.WithLogOutput(logWriter))
		if err != nil {
			slog.Error("Error starting dagger", "error", err)

			if isDockerDaemonError(err) {
				handleDockerDaemonError()
			}

			os.Exit(1)
		}
		defer dag.Close()

		return mcpserver.RunStdioServer(ctx, dag)
	},
}

func init() {
	rootCmd.AddCommand(stdioCmd)
}
