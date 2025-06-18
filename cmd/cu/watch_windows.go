//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch git log output",
	Long:  `Watch the following git log command every second: 'git log --color=always --remotes=container-use --oneline --graph --decorate'.`,
	RunE: func(app *cobra.Command, _ []string) error {
		ctx := app.Context()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		// Run once immediately
		if err := runGitLog(ctx); err != nil {
			return err
		}

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				if err := runGitLog(ctx); err != nil {
					return err
				}
			}
		}
	},
}

func runGitLog(ctx context.Context) error {
	// Clear screen (Windows cmd equivalent)
	clearCmd := exec.CommandContext(ctx, "cmd", "/c", "cls")
	clearCmd.Stdout = os.Stdout
	clearCmd.Run() // Ignore errors for clearing screen

	// Run git log command
	cmd := exec.CommandContext(ctx, "git", "log", "--color=always", "--remotes=container-use", "--oneline", "--graph", "--decorate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running git log: %v\n", err)
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
