//go:build unix

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func execDagger(args []string) error {
	daggerBin, err := exec.LookPath("dagger")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("dagger is not installed. Please install it from https://docs.dagger.io/install/")
		}
		return fmt.Errorf("failed to look up dagger binary: %w", err)
	}
	return syscall.Exec(daggerBin, append([]string{"dagger", "run"}, args...), os.Environ())
}
