//go:build !windows

package main

import (
	"syscall"
)

func execDaggerRun(daggerBin string, args []string, env []string) error {
	return syscall.Exec(daggerBin, args, env)
}
