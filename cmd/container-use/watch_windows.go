//go:build windows

package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch environment activity in real-time",
	Long: `Continuously display environment activity as agents work.
Shows new commits and environment changes updated every second.
Press Ctrl+C to stop watching.`,
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

		// Enter alternate screen buffer and hide cursor
		fmt.Print("\x1b[?1049h\x1b[?25l")
		defer fmt.Print("\x1b[?25h\x1b[?1049l") // restore screen + show cursor

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

// runGitLogWindows executes the git log command with output matching Unix watch format
func runGitLogWindows(ctx context.Context) error {
	var buf bytes.Buffer

	// Clear screen and move cursor to home position
	buf.WriteString("\x1b[H\x1b[J")

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Format timestamp like Unix watch (Day Month DD HH:MM:SS YYYY)
	timestamp := time.Now().Format("Mon Jan 2 15:04:05 2006")

	// Create header that matches Unix watch format exactly
	gitCommand := "git log --color=always --remotes=container-use --oneline --graph --decorate"
	headerLine := fmt.Sprintf("Every 1.0s: %s", gitCommand)

	// Get terminal width, fallback to 80 if unable to determine
	terminalWidth := 80
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 {
		terminalWidth = width
	}

	// Calculate spaces needed to right-align the hostname and timestamp
	rightPart := fmt.Sprintf("%s: %s", hostname, timestamp)
	spacesNeeded := terminalWidth - len(headerLine) - len(rightPart)

	// Write the header line with responsive spacing
	if spacesNeeded >= 1 {
		// Enough space to fit on one line - right align
		buf.WriteString(headerLine + strings.Repeat(" ", spacesNeeded) + rightPart + "\n")
	} else {
		// Not enough space - wrap to next line with minimum 1 space
		buf.WriteString(headerLine + " " + rightPart + "\n")
	}
	buf.WriteString("\n") // Empty line after header to match Unix watch format

	// Run git log command with same arguments as Unix version
	cmd := exec.CommandContext(ctx, "git", "log",
		"--color=always",
		"--remotes=container-use",
		"--oneline",
		"--graph",
		"--decorate")

	// Create pipe to capture output
	pr, pw, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}

	cmd.Stdout = pw
	cmd.Stderr = pw

	// Start the command
	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		return fmt.Errorf("failed to start git log: %w", err)
	}

	// Close write end so we can read
	pw.Close()

	// Read all output into buffer
	scanner := bufio.NewScanner(pr)
	for scanner.Scan() {
		buf.WriteString(scanner.Text() + "\n")
	}

	pr.Close()

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git log failed: %w", err)
	}

	// Output everything at once for smooth rendering
	os.Stdout.Write(buf.Bytes())

	return nil
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
