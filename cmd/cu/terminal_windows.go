//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func execDagger(args []string) error {
	daggerBin, err := exec.LookPath("dagger")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("dagger is not installed. Please install it from https://docs.dagger.io/install/")
		}
		return fmt.Errorf("failed to look up dagger binary: %w", err)
	}

	cmd := exec.Command(daggerBin, append([]string{"run"}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err = cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		return err
	}

	os.Exit(0)
	return nil
}
