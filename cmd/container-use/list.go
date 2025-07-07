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
	Short: "List all environments",
	Long: `Display all active environments with their IDs, titles, and timestamps.
Use -q for environment IDs only, useful for scripting.`,
	RunE: func(app *cobra.Command, _ []string) error {
		ctx := app.Context()
		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return err
		}
		envInfos, err := repo.List(ctx)
		if err != nil {
			return err
		}
		if quiet, _ := app.Flags().GetBool("quiet"); quiet {
			for _, envInfo := range envInfos {
				fmt.Println(envInfo.ID)
			}
			return nil
		}

		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tTITLE\tCREATED\tUPDATED")

		defer tw.Flush()
		for _, envInfo := range envInfos {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", envInfo.ID, truncate(app, envInfo.State.Title, 40), humanize.Time(envInfo.State.CreatedAt), humanize.Time(envInfo.State.UpdatedAt))
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
