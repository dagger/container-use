package main

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShowCommandIsTopLevelAliasForConfigShow(t *testing.T) {
	showCmd, _, err := rootCmd.Find([]string{"show"})
	require.NoError(t, err)

	configShowCmd, _, err := rootCmd.Find([]string{"config", "show"})
	require.NoError(t, err)
	require.NotNil(t, showCmd)
	require.NotNil(t, configShowCmd)

	assert.Equal(t, "show", showCmd.Name())
	assert.Equal(t, configShowCmd.Use, showCmd.Use)
	assert.Equal(t, configShowCmd.Short, showCmd.Short)
	assert.Equal(t, configShowCmd.Long, showCmd.Long)
	assert.Equal(t, `# Show the default environment configuration
container-use show

# Show the configuration for a specific environment
container-use show my-env`, showCmd.Example)
	assert.Equal(t, reflect.ValueOf(configShowCmd.RunE).Pointer(), reflect.ValueOf(showCmd.RunE).Pointer())
	assert.Equal(t, reflect.ValueOf(configShowCmd.Args).Pointer(), reflect.ValueOf(showCmd.Args).Pointer())
	assert.Equal(t, reflect.ValueOf(configShowCmd.ValidArgsFunction).Pointer(), reflect.ValueOf(showCmd.ValidArgsFunction).Pointer())
	assert.NotNil(t, configShowCmd.Flags().Lookup("json"))
	assert.NotNil(t, showCmd.Flags().Lookup("json"))
}

func TestShowCommandVisibleInRootHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	oldOut := rootCmd.OutOrStdout()
	oldErr := rootCmd.ErrOrStderr()
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer rootCmd.SetOut(oldOut)
	defer rootCmd.SetErr(oldErr)

	err := rootCmd.Help()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "show")
	assert.Contains(t, output, "Show environment configuration")
}
