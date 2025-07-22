package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/dagger/container-use/repository"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

var followCmd = &cobra.Command{
	Use:   "follow <env>",
	Short: "Checkout environment and continuously pull changes",
	Long: `Checkout an environment's branch locally and continuously pull changes from the remote.
This command first performs a checkout operation, then continuously monitors for
changes in the environment's remote branch and pulls them automatically.

Uses file system watching on the git refs directory for near-immediate response
when the environment is updated. Falls back to periodic polling for reliability.
Press Ctrl+C to stop following.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: suggestEnvironments,
	Example: `# Follow environment changes
container-use follow fancy-mallard

# Follow with custom branch name
container-use follow fancy-mallard -b my-review-branch

# Follow with custom fallback interval
container-use follow fancy-mallard --fallback-interval 30s`,
	RunE: func(app *cobra.Command, args []string) error {
		ctx := app.Context()
		envID := args[0]

		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return err
		}

		branchName, err := app.Flags().GetString("branch")
		if err != nil {
			return err
		}

		fallbackInterval, err := app.Flags().GetDuration("fallback-interval")
		if err != nil {
			return err
		}

		// First, perform the checkout
		branch, err := repo.Checkout(ctx, envID, branchName)
		if err != nil {
			return err
		}

		slog.Info("switched to branch", "branch", branch)
		slog.Info("following environment for changes", "env-id", envID, "fallback-interval", fallbackInterval)
		slog.Info("press Ctrl+C to stop following")

		// Set up signal handling for graceful shutdown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		// Start following with file watching
		return followWithFileWatching(ctx, repo, envID, fallbackInterval, sigCh)
	},
}

func followWithFileWatching(ctx context.Context, repo *repository.Repository, envID string, fallbackInterval time.Duration, sigCh chan os.Signal) error {
	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Warn("failed to create file watcher, falling back to polling", "error", err)
		return followWithPolling(ctx, repo, envID, fallbackInterval, sigCh)
	}
	defer watcher.Close()

	// Path to the remote ref file and its parent directory
	refPath := filepath.Join(repo.SourcePath(), ".git", "refs", "remotes", "container-use", envID)
	refDir := filepath.Dir(refPath)

	// Watch the ref directory to catch atomic writes
	err = watcher.Add(refDir)
	if err != nil {
		slog.Warn("failed to watch ref directory, falling back to polling", "ref-dir", refDir, "error", err)
		return followWithPolling(ctx, repo, envID, fallbackInterval, sigCh)
	}

	slog.Debug("starting file watcher for environment", "env-id", envID, "ref-path", refPath)

	// Fallback ticker in case file watching misses something
	fallbackTicker := time.NewTicker(fallbackInterval)
	defer fallbackTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-sigCh:
			slog.Info("stopping follow")
			return nil
		case event := <-watcher.Events:
			slog.Debug("file event received", "op", event.Op, "name", event.Name)
			// Check if the event is for our specific ref file
			if event.Name == refPath {
				slog.Debug("triggering pull for ref file event", "op", event.Op)
				if err := pullChanges(ctx, repo, envID); err != nil {
					slog.Error("failed to pull changes", "error", err)
				}
			} else {
				slog.Debug("ignoring event on unrelated file", "name", event.Name)
			}
		case err := <-watcher.Errors:
			slog.Error("file watcher error", "error", err)
		case <-fallbackTicker.C:
			slog.Debug("fallback check triggered")
			// Fallback check in case file watching missed something
			if err := pullChanges(ctx, repo, envID); err != nil {
				slog.Error("failed to pull changes during fallback", "error", err)
			}
		}
	}
}

func followWithPolling(ctx context.Context, repo *repository.Repository, envID string, interval time.Duration, sigCh chan os.Signal) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-sigCh:
			slog.Info("stopping follow")
			return nil
		case <-ticker.C:
			if err := pullChanges(ctx, repo, envID); err != nil {
				slog.Error("failed to pull changes", "error", err)
			}
		}
	}
}

func pullChanges(ctx context.Context, repo *repository.Repository, envID string) error {
	slog.Debug("fetching changes from remote", "env-id", envID)
	// First, fetch the latest changes from the container-use remote
	_, err := repository.RunGitCommand(ctx, repo.SourcePath(), "fetch", "container-use", envID)
	if err != nil {
		return fmt.Errorf("failed to fetch changes: %w", err)
	}
	slog.Debug("fetch completed successfully")

	// Check if there are any new changes to pull
	remoteRef := fmt.Sprintf("container-use/%s", envID)
	counts, err := repository.RunGitCommand(ctx, repo.SourcePath(), "rev-list", "--left-right", "--count", fmt.Sprintf("HEAD...%s", remoteRef))
	if err != nil {
		return fmt.Errorf("failed to check for changes: %w", err)
	}

	// Parse the output to determine if there are changes
	parts := strings.Split(strings.TrimSpace(counts), "\t")
	if len(parts) != 2 {
		return fmt.Errorf("unexpected git rev-list output: %s", counts)
	}
	aheadCount, behindCount := parts[0], parts[1]

	// If we're behind, pull the changes
	if behindCount != "0" {
		if aheadCount == "0" {
			// Fast-forward merge
			_, err = repository.RunGitCommand(ctx, repo.SourcePath(), "merge", "--ff-only", remoteRef)
			if err != nil {
				return fmt.Errorf("failed to fast-forward merge: %w", err)
			}
			slog.Info("pulled new commits from environment", "commits", behindCount, "env-id", envID)
		} else {
			// Local changes exist, notify user
			slog.Warn("environment has new commits but local branch is ahead, manual merge required", "env-id", envID, "remote-commits", behindCount, "local-commits", aheadCount)
		}
	}

	return nil
}

func init() {
	followCmd.Flags().StringP("branch", "b", "", "Local branch name to use")
	followCmd.Flags().DurationP("fallback-interval", "i", 30*time.Second, "Fallback polling interval (e.g., 30s, 1m)")
	rootCmd.AddCommand(followCmd)
}
