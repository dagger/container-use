package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveEnvironmentID(t *testing.T) {
	t.Run("WithProvidedArgs", func(t *testing.T) {
		// When args are provided, should return the first arg directly
		ctx := context.Background()
		args := []string{"test-env"}

		// Don't need a real repository for this test
		envID, err := resolveEnvironmentID(ctx, nil, args)
		require.NoError(t, err)
		assert.Equal(t, "test-env", envID)
	})

	t.Run("WithMultipleArgs", func(t *testing.T) {
		// When multiple args are provided, should return the first arg
		ctx := context.Background()
		args := []string{"test-env", "other-arg"}

		envID, err := resolveEnvironmentID(ctx, nil, args)
		require.NoError(t, err)
		assert.Equal(t, "test-env", envID)
	})

	// Note: Testing with empty args requires a real repository and is tested
	// in the repository integration tests and environment integration tests
}
