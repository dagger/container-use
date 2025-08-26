package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "container-use",
		Short: "Containerized environments for coding agents",
		Long: `Container Use creates isolated development environments for AI agents.
Each environment runs in its own container with dedicated git branches.`,
	}
)

func main() {
	ctx := context.Background()
	setupSignalHandling()

	if err := setupLogger(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logger: %v\n", err)
		os.Exit(1)
	}

	// FIXME(aluzzardi): `fang` misbehaves with the `stdio` command.
	// It hangs on Ctrl-C. Traced the hang back to `lipgloss.HasDarkBackground(os.Stdin, os.Stdout)`
	// I'm assuming it's not playing nice the mcpserver listening on stdio.
	if len(os.Args) > 1 && os.Args[1] == "stdio" {
		if err := rootCmd.ExecuteContext(ctx); err != nil {
			os.Exit(1)
		}
		return
	}

	if err := fang.Execute(
		ctx,
		rootCmd,
		fang.WithVersion(version),
		fang.WithCommit(commit),
		fang.WithNotifySignal(getNotifySignals()...),
	); err != nil {
		os.Exit(1)
	}
}

func suggestEnvironments(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()

	repo, err := repository.Open(ctx, ".")
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// Use the standard List method - it's already parallelized and works correctly
	envs, err := repo.List(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// If no environments found, return directive that prevents fallback to file completion
	if len(envs) == 0 {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	// Extract IDs from environment info
	ids := make([]string, len(envs))
	for i, env := range envs {
		ids[i] = env.ID
	}

	return ids, cobra.ShellCompDirectiveNoFileComp
}
