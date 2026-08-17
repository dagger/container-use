package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerminalDaggerRunArgs(t *testing.T) {
	args := terminalDaggerRunArgs([]string{"container-use", "terminal", "fancy-mammal"}, "/workspace")

	assert.Equal(
		t,
		[]string{"dagger", "run", "--source", "/workspace", "container-use", "terminal", "fancy-mammal"},
		args,
	)
}

func TestTerminalSourcePathReturnsAbsolutePath(t *testing.T) {
	sourcePath, err := terminalSourcePath(".")
	require.NoError(t, err)

	assert.True(t, filepath.IsAbs(sourcePath))
	assert.Equal(t, sourcePath, filepath.Clean(sourcePath))
}
