package main

import (
	"encoding/json"
	"fmt"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:               "inspect [<env>]",
	Short:             "Inspect an environment",
	Long:              "This is an internal command used by the CLI to inspect an environment. It is not meant to be used by users.",
	Args:              cobra.MaximumNArgs(1),
	Hidden:            true,
	ValidArgsFunction: suggestEnvironments,
	RunE: func(app *cobra.Command, args []string) error {
		ctx := app.Context()
		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return err
		}

		envID, err := resolveEnvironmentID(ctx, repo, args)
		if err != nil {
			return err
		}

		envInfo, err := repo.Info(ctx, envID)
		if err != nil {
			return err
		}

		envInfo.State.Container = ""
		out, err := json.MarshalIndent(envInfo, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(out))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}
