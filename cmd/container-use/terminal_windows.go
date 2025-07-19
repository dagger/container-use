//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func execDaggerRun(daggerBin string, args []string, env []string) error {
	cmd := exec.Command(daggerBin, args...)
	cmd.Args = append([]string{"dagger", "run"}, os.Args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute dagger run: %w", err)
	}

	// On Windows, we can't replace the current process, so we exit
	os.Exit(0)
	return nil
}
