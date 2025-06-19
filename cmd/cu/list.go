package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/dagger/container-use/repository"
	"github.com/dustin/go-humanize"
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
		if quiet, _ := app.Flags().GetBool("quiet"); quiet {
			for _, env := range envs {
				fmt.Println(env.ID)
			}
			return nil
		}

		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tTITLE\tCREATED\tUPDATED")

		defer tw.Flush()
		for _, env := range envs {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", env.ID, truncate(app, env.State.Title, 40), humanize.Time(env.State.CreatedAt), humanize.Time(env.State.UpdatedAt))
		}
		return nil
	},
}

func truncate(app *cobra.Command, s string, max int) string {
	if noTrunc, _ := app.Flags().GetBool("no-trunc"); noTrunc {
		return s
	}
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}

func init() {
	listCmd.Flags().BoolP("quiet", "q", false, "Display only environment IDs")
	listCmd.Flags().BoolP("no-trunc", "", false, "Don't truncate output")
	rootCmd.AddCommand(listCmd)
}
