package environment

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateMarshalUnmarshal(t *testing.T) {
	state := &State{
		Title:     "test env",
		Container: "container-id-123",
		Config:    DefaultConfig(),
	}
	data, err := state.Marshal()
	require.NoError(t, err)

	var loaded State
	require.NoError(t, loaded.Unmarshal(data))
	assert.Equal(t, "test env", loaded.Title)
	assert.Equal(t, "container-id-123", loaded.Container)
	assert.Equal(t, "ubuntu:24.04", loaded.Config.BaseImage)
}

func TestStateUnmarshalInvalidJSON(t *testing.T) {
	var state State
	err := state.Unmarshal([]byte("not json"))
	assert.Error(t, err)
}

func TestStateUnmarshalLegacy(t *testing.T) {
	legacy := []byte(`[
		{"version": 1, "name": "first", "explanation": "", "state": "old-state-1", "created_at": "2024-01-01T00:00:00Z"},
		{"version": 2, "name": "latest", "explanation": "", "state": "latest-state", "created_at": "2024-06-01T00:00:00Z"}
	]`)

	var state State
	require.NoError(t, state.Unmarshal(legacy))
	assert.Equal(t, "latest-state", state.Container)
	assert.WithinDuration(t, time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), state.UpdatedAt, 0)
}

func TestLegacyStateLatest(t *testing.T) {
	ls := legacyState{
		{Version: 1, State: "s1"},
		{Version: 2, State: "s2"},
	}
	assert.Equal(t, 2, ls.Latest().Version)
	assert.Equal(t, 2, ls.LatestVersion())
	assert.Equal(t, "s1", ls.Get(1).State)
	assert.Nil(t, ls.Get(99))
}

func TestLegacyStateLatestEmpty(t *testing.T) {
	var ls legacyState
	assert.Nil(t, ls.Latest())
	assert.Equal(t, 0, ls.LatestVersion())
}
