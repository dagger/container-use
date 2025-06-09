package main

import (
	"time"

	"github.com/spf13/cobra"
	watch "github.com/tiborvass/go-watch"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch git log output",
	Long:  `Watch the following git log command every second: 'git log --color=always --remotes=container-use --oneline --graph --decorate'.`,
	RunE: func(app *cobra.Command, _ []string) error {
		w := watch.Watcher{Interval: time.Second}
		w.Watch(app.Context(), "git", "log", "--color=always", "--remotes=container-use", "--oneline", "--graph", "--decorate")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
