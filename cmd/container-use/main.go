package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/dagger/container-use/repository"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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

func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Default to 120 columns if we can't detect terminal size -- this seems common
		return 120
	}
	return width
}

// calculateMaxTitleLength calculates the maximum length for title truncation
// based on terminal width, leaving room for environment ID, description format, and padding
func calculateMaxTitleLength(terminalWidth int) int {
	// Format: "env-id	description: title (updated time ago)"
	// We need to account for:
	// - Environment ID (typically 8-15 chars like "adapted-tetra")
	// - Tab separator (1 char)
	// - Description prefix/suffix like " (updated " and ")"
	// - Time string like "2 hours ago" (typically 5-15 chars)
	// - Some padding for safety

	const (
		avgEnvIDLength = 12 // typical environment ID length
		tabSeparator   = 1  // tab character
		descSuffix     = 11 // " (updated "
		avgTimeLength  = 10 // "2 hours ago"
		closeParen     = 1  // ")"
		padding        = 5  // safety padding
	)

	usedSpace := avgEnvIDLength + tabSeparator + descSuffix + avgTimeLength + closeParen + padding
	maxTitleLength := terminalWidth - usedSpace

	// Ensure we have a reasonable minimum and maximum
	if maxTitleLength < 10 {
		return 10 // minimum readable length
	}
	if maxTitleLength > 100 {
		return 100 // reasonable maximum
	}

	return maxTitleLength
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

	// Create completions with descriptions showing title and update time
	terminalWidth := getTerminalWidth()
	maxTitleLength := calculateMaxTitleLength(terminalWidth)

	completions := make([]string, len(envs))
	for i, env := range envs {
		title := env.State.Title
		if len(title) > maxTitleLength {
			title = title[:maxTitleLength] + "…"
		}
		description := fmt.Sprintf("%s (updated %s)", title, humanize.Time(env.State.UpdatedAt))
		completions[i] = cobra.CompletionWithDesc(env.ID, description)
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
