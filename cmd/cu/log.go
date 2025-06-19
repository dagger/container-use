package main

import (
	"os"
	"os/exec"
	"strings"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log <env>",
	Short: "Show the log for an environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(app *cobra.Command, args []string) error {
		ctx := app.Context()

		// Ensure we're in a git repository
		if _, err := repository.Open(ctx, "."); err != nil {
			return err
		}

		env := args[0]
		// prevent accidental single quotes to mess up command
		env = strings.Trim(env, "'")
		cmd := exec.CommandContext(app.Context(), "git", "log", "--patch", "container-use/"+env)
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout

		return cmd.Run()
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
}
