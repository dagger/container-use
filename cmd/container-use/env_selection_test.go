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

	// Note: Testing with no args requires a real repository and is tested
	// in environment/integration/environment_selection_test.go
}
