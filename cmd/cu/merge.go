package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var mergeCmd = &cobra.Command{
	Use:   "merge <env>",
	Short: "Merges an environment into the current git branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(app *cobra.Command, args []string) error {
		env := args[0]
		ctx := app.Context()
		err := exec.Command("git", "stash", "--include-untracked", "-q").Run()
		if err == nil {
			defer exec.Command("git", "stash", "pop", "-q").Run()
		}
		cmd := exec.CommandContext(ctx, "git", "merge", "-m", fmt.Sprintf("chore(merge): merge environment %s", env), "--", "container-use/"+env)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		return cmd.Run()
	},
}

func init() {
	rootCmd.AddCommand(mergeCmd)
}
