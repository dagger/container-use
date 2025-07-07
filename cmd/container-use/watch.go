package main

import (
	"time"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
	watch "github.com/tiborvass/go-watch"
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

		w := watch.Watcher{Interval: time.Second}
		w.Watch(app.Context(), "git", "log", "--color=always", "--remotes=container-use", "--oneline", "--graph", "--decorate")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
