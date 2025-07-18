//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch environment activity in real-time",
	Long: `Continuously display environment activity as agents work.
Shows new commits and environment changes updated every second.
Press Ctrl+C to stop watching.

This Windows-compatible implementation updates every second and clears
the screen between updates for a clean display.`,
	Example: `# Watch all environment activity
container-use watch

# Monitor agents while they work
container-use watch`,
	RunE: func(app *cobra.Command, _ []string) error {
		ctx := app.Context()

		// Ensure we're in a git repository
		if _, err := repository.Open(ctx, "."); err != nil {
			return err
		}

		// Display initial header
		fmt.Printf("Container-use watch on %s - Press Ctrl+C to stop\n", runtime.GOOS)
		fmt.Println("Watching: git log --remotes=container-use --oneline --graph --decorate")
		fmt.Println(strings.Repeat("=", 70))

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		// Run once immediately
		if err := runGitLogWindows(ctx); err != nil {
			return err
		}

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				if err := runGitLogWindows(ctx); err != nil {
					// Don't exit on git errors, just display them and continue
					fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
				}
			}
		}
	},
}

// runGitLogWindows executes the git log command optimized for Windows
func runGitLogWindows(ctx context.Context) error {
	// Clear screen using Windows cmd
	clearCmd := exec.CommandContext(ctx, "cmd", "/c", "cls")
	clearCmd.Stdout = os.Stdout
	clearCmd.Run() // Ignore errors for clearing screen

	// Display timestamp header
	fmt.Printf("Last updated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println(strings.Repeat("-", 50))

	// Run git log command (without --color=always for Windows compatibility)
	cmd := exec.CommandContext(ctx, "git", "log",
		"--remotes=container-use",
		"--oneline",
		"--graph",
		"--decorate",
		"--max-count=20") // Limit output to prevent overwhelming the screen

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git log failed: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
