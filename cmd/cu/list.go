package main

import (
	"fmt"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List environments",
	Long:  `List environments filtering the git remotes`,
	RunE: func(app *cobra.Command, _ []string) error {
		ctx := app.Context()
		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return err
		}
		envs, err := repo.List(ctx)
		if err != nil {
			return err
		}
		for _, env := range envs {
			fmt.Println(env)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
