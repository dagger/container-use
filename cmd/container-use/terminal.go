package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"dagger.io/dagger"
	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var terminalCmd = &cobra.Command{
	Use:   "terminal [<env>]",
	Short: "Get a shell inside an environment's container",
	Long: `Open an interactive terminal in the exact container environment the agent used. Perfect for debugging, testing, or hands-on exploration.

If no environment is specified, automatically selects from environments 
that are descendants of the current HEAD.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: suggestEnvironments,
	Example: `# Drop into environment's container
container-use terminal fancy-mallard

# Debug agent's work interactively
container-use terminal backend-api

# Auto-select environment
container-use terminal`,
	RunE: func(app *cobra.Command, args []string) error {
		ctx := app.Context()

		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return err
		}

		// FIXME(aluzzardi): This is a hack to make sure we're wrapped in `dagger run` since `Terminal()` only works with the CLI.
		// If not, it will auto-wrap this command in a `dagger run`.
		if _, ok := os.LookupEnv("DAGGER_SESSION_TOKEN"); !ok {
			daggerBin, err := exec.LookPath("dagger")
			if err != nil {
				if errors.Is(err, exec.ErrNotFound) {
					return fmt.Errorf("dagger is not installed. Please install it from https://docs.dagger.io/install/")
				}
				return fmt.Errorf("failed to look up dagger binary: %w", err)
			}
			return execDaggerRun(daggerBin, append([]string{"dagger", "run"}, os.Args...), os.Environ())
		}

		dag, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
		if err != nil {
			if isDockerDaemonError(err) {
				handleDockerDaemonError()
			}
			return fmt.Errorf("failed to connect to dagger: %w", err)
		}
		defer dag.Close()

		envID, err := resolveEnvironmentID(ctx, repo, args)
		if err != nil {
			return err
		}

		env, err := repo.Get(ctx, dag, envID)
		if err != nil {
			return err
		}

		return env.Terminal(ctx)
	},
}

func init() {
	rootCmd.AddCommand(terminalCmd)
}
