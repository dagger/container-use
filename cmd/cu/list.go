package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List environments",
	Long:  `List environments filtering the git remotes`,
	RunE: func(app *cobra.Command, _ []string) error {
		// Check if we're in a git repository
		checkCmd := exec.CommandContext(app.Context(), "git", "rev-parse", "--is-inside-work-tree")
		if err := checkCmd.Run(); err != nil {
			return fmt.Errorf("cu list only works within git repository, no repo found (or any of the parent directories): .git")
		}

		cmd := exec.CommandContext(app.Context(), "bash", "-c", "git branch -r | grep 'container-use/.*/' | cut -d/ -f2-")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		return cmd.Run()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
