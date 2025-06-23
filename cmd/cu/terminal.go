package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"dagger.io/dagger"
	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var terminalCmd = &cobra.Command{
	Use:   "terminal <env>",
	Short: "Drop a terminal into an environment",
	Long:  `Create a container with the same state as the agent for a given branch or commmit.`,
	Args:  cobra.ExactArgs(1),
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
			return syscall.Exec(daggerBin, append([]string{"dagger", "run"}, os.Args...), os.Environ())
		}

		dag, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
		if err != nil {
			return fmt.Errorf("failed to connect to dagger: %w", err)
		}
		defer dag.Close()
		env, err := repo.Get(ctx, dag, args[0])
		if err != nil {
			return err
		}

		return env.Terminal(ctx)
	},
}
