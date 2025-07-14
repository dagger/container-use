package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveEnvironmentID(t *testing.T) {
	t.Run("WithSingleArg", func(t *testing.T) {
		// When one arg is provided, should return it directly
		ctx := context.Background()
		args := []string{"test-env"}

		envID, err := resolveEnvironmentID(ctx, nil, args)
		require.NoError(t, err)
		assert.Equal(t, "test-env", envID)
	})

	t.Run("WithMultipleArgs", func(t *testing.T) {
		// When multiple args are provided, should return an error
		ctx := context.Background()
		args := []string{"test-env", "other-arg"}

		_, err := resolveEnvironmentID(ctx, nil, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too many arguments")
	})

	t.Run("WithNoArgs", func(t *testing.T) {
		// When no args are provided, should try to resolve from repository
		// This will fail with a nil repository but exercises the code path
		ctx := context.Background()
		args := []string{}

		_, err := resolveEnvironmentID(ctx, nil, args)
		assert.Error(t, err)
		// Should not be the "too many arguments" error
		assert.NotContains(t, err.Error(), "too many arguments")
	})

	// Note: Full integration testing with repository logic is in 
	// environment/integration/environment_selection_test.go
}