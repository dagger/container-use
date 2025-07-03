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

	"github.com/charmbracelet/fang"
	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "container-use",
		Short: "Containerized environments for coding agents",
		Long: `Container Use creates isolated development environments for AI agents.
Each environment runs in its own container with dedicated git branches.`,
	}
)

func main() {
	ctx := context.Background()
	sigusrCh := make(chan os.Signal, 1)
	signal.Notify(sigusrCh, syscall.SIGUSR1)
	go handleSIGUSR(sigusrCh)

	if err := setupLogger(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logger: %v\n", err)
		os.Exit(1)
	}

	// FIXME(aluzzardi): `fang` misbehaves with the `stdio` command.
	// It hangs on Ctrl-C. Traced the hang back to `lipgloss.HasDarkBackground(os.Stdin, os.Stdout)`
	// I'm assuming it's not playing nice the mcpserver listening on stdio.
	if len(os.Args) > 1 && os.Args[1] == "stdio" {
		if err := rootCmd.ExecuteContext(ctx); err != nil {
			os.Exit(1)
		}
		return
	}

	if err := fang.Execute(
		ctx,
		rootCmd,
		fang.WithVersion(version),
		fang.WithCommit(commit),
		fang.WithNotifySignal(os.Interrupt, os.Kill, syscall.SIGTERM),
	); err != nil {
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
