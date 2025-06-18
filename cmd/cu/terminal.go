package main

import (
	"log/slog"
	"os"

	"dagger.io/dagger"
	"github.com/dagger/container-use/environment"
	"github.com/spf13/cobra"
)

var terminalCmd = &cobra.Command{
	Use:   "terminal <env>",
	Short: "Drop a terminal into an environment",
	Long:  `Create a container with the same state as the agent for a given branch or commmit.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(app *cobra.Command, args []string) error {
		ctx := app.Context()

		// FIXME(aluzzardi): This is a hack to make sure we're wrapped in `dagger run` since `Terminal()` only works with the CLI.
		// If not, it will auto-wrap this command in a `dagger run`.
		if _, ok := os.LookupEnv("DAGGER_SESSION_TOKEN"); !ok {
			return execDagger(os.Args)
		}

		dag, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
		if err != nil {
			slog.Error("Error starting dagger", "error", err)
			os.Exit(1)
		}
		defer dag.Close()
		environment.Initialize(dag)

		env, err := environment.Open(ctx, "opening terminal", ".", args[0])
		if err != nil {
			return err
		}

		return env.Terminal(ctx)
	},
}
