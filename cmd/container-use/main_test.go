package main

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateMaxTitleLength(t *testing.T) {
	assert.Equal(t, 10, calculateMaxTitleLength(10))
	assert.Equal(t, 100, calculateMaxTitleLength(200))

	// Typical terminal width should yield a sensible middle value
	mid := calculateMaxTitleLength(120)
	assert.Greater(t, mid, 10)
	assert.LessOrEqual(t, mid, 100)
}

func TestTruncate(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("no-trunc", false, "")

	assert.Equal(t, "hello", truncate(cmd, "hello", 10))
	assert.Equal(t, "hel…", truncate(cmd, "hello", 3))
	assert.Equal(t, "", truncate(cmd, "", 5))

	cmd.Flags().Set("no-trunc", "true")
	assert.Equal(t, "hello world", truncate(cmd, "hello world", 3))
}

func TestSuggestEnvironmentsNoRepo(t *testing.T) {
	cmd := &cobra.Command{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd.SetContext(ctx)

	orig, err := os.Getwd()
	require.NoError(t, err)
	tmp := t.TempDir()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(orig) })

	completions, directive := suggestEnvironments(cmd, nil, "")
	assert.Empty(t, completions)
	assert.Equal(t, cobra.ShellCompDirectiveError, directive)
}

func TestCompletionDescriptionFormat(t *testing.T) {
	// CompletionWithDesc is a Cobra helper; verify the format it produces.
	completion := cobra.CompletionWithDesc("env-id", "title (updated 2 hours ago)")
	assert.True(t, strings.HasPrefix(completion, "env-id\t"))
	assert.Contains(t, completion, "title (updated 2 hours ago)")
}
