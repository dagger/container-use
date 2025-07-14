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

	t.Run("WithEmptyArgs", func(t *testing.T) {
		// When no args are provided, should try to resolve from repository
		// This test verifies the function calls into the repository logic
		// (The actual repository logic is tested in repository package)
		ctx := context.Background()
		args := []string{}

		// This will fail because we don't have a real repository
		// but it verifies the code path is exercised
		_, err := resolveEnvironmentID(ctx, nil, args)
		assert.Error(t, err)
	})
}