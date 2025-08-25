package main

import (
	"fmt"
	"time"

	"github.com/dagger/container-use/repository"
	"github.com/karrick/tparse"
	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Delete environments older than specified age",
	Long: `Delete environments that haven't been updated within the specified time period.
This permanently removes old environments and their associated resources including
branches and container state. By default, environments older than 1 week are pruned.

Use --dry-run to see what would be deleted without actually deleting anything.
Use --before to configure the age threshold (e.g., 24h, 3d, 2w, 1mo).`,
	Example: `# Prune environments older than 1 week (default)
container-use prune

# Prune environments older than 3 days
container-use prune --before 3d

# See what would be pruned without deleting
container-use prune --dry-run

# Prune environments older than 2 weeks
container-use prune --before 2w`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		before, _ := cmd.Flags().GetString("before")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return fmt.Errorf("failed to open repository: %w", err)
		}

		var duration time.Duration
		if before == "" {
			duration = 7 * 24 * time.Hour
		} else {
			targetTime, err := tparse.ParseNow(time.RFC3339, "now-"+before)
			if err != nil {
				return fmt.Errorf("invalid --before format: %w", err)
			}
			duration = time.Since(targetTime)
		}

		envs, err := repo.List(ctx)
		if err != nil {
			return fmt.Errorf("failed to list environments: %w", err)
		}

		if len(envs) == 0 {
			fmt.Println("No environments found.")
			return nil
		}

		cutoff := time.Now().Add(-duration)
		var envsToPrune []string

		for _, env := range envs {
			if env.State.UpdatedAt.Before(cutoff) {
				envsToPrune = append(envsToPrune, env.ID)
			}
		}

		if len(envsToPrune) == 0 {
			fmt.Printf("No environments older than %s found.\n", duration)
			return nil
		}

		if dryRun {
			fmt.Printf("Would prune %d environment(s) older than %s:\n", len(envsToPrune), duration)
			for _, envID := range envsToPrune {
				fmt.Printf("  - %s\n", envID)
			}
			return nil
		}

		fmt.Printf("Pruning %d environment(s) older than %s...\n", len(envsToPrune), duration)

		var deletedCount int
		for _, envID := range envsToPrune {
			if err := repo.Delete(ctx, envID); err != nil {
				fmt.Printf("Failed to delete environment '%s': %v\n", envID, err)
			} else {
				fmt.Printf("Environment '%s' deleted successfully.\n", envID)
				deletedCount++
			}
		}

		fmt.Printf("Successfully deleted %d environment(s).\n", deletedCount)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)
	pruneCmd.Flags().String("before", "1w", "Delete environments older than this duration (e.g., 24h, 3d, 2w, 1mo)")
	pruneCmd.Flags().Bool("dry-run", false, "Show what would be pruned without actually deleting")
}
