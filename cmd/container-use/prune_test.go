package main

import (
	"testing"
)

func TestPruneCommand(t *testing.T) {
	// Basic smoke test to ensure prune command is registered and can be built
	if pruneCmd == nil {
		t.Error("pruneCmd should not be nil")
	}

	if pruneCmd.Use != "prune" {
		t.Errorf("expected command name 'prune', got %q", pruneCmd.Use)
	}

	// Test that flags are registered
	beforeFlag := pruneCmd.Flag("before")
	if beforeFlag == nil {
		t.Error("--before flag should be registered")
	}

	dryRunFlag := pruneCmd.Flag("dry-run")
	if dryRunFlag == nil {
		t.Error("--dry-run flag should be registered")
	}
}