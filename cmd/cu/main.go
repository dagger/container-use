package main

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "cu",
		Short: "Containerized environments for coding agents",
		Long: `Container Use creates isolated development environments for AI agents.
Each environment runs in its own container with dedicated git branches.`,
	}
)

func main() {
	sigusrCh := make(chan os.Signal, 1)
	signal.Notify(sigusrCh, syscall.SIGUSR1)

	go handleSIGUSR(sigusrCh)

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	if err := setupLogger(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logger: %v\n", err)
		os.Exit(1)
	}

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func handleSIGUSR(sigusrCh <-chan os.Signal) {
	for sig := range sigusrCh {
		if sig == syscall.SIGUSR1 {
			dumpStacks()
		}
	}
}

func dumpStacks() {
	buf := make([]byte, 1<<20) // 1MB buffer
	n := runtime.Stack(buf, true)
	io.MultiWriter(logWriter, os.Stderr).Write(buf[:n])
}

func suggestEnvironments(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()

	repo, err := repository.Open(ctx, ".")
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	envs, err := repo.List(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	ids := []string{}
	for _, e := range envs {
		ids = append(ids, e.ID)
	}
	return ids, cobra.ShellCompDirectiveKeepOrder
}
