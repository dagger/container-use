package environment

import (
	"encoding/json"
	"fmt"
	"time"

	"dagger.io/dagger"
)

type State struct {
	Container string `json:"container"`
}

func migrateLegacyState(state []byte) (*State, error) {
	var history legacyState
	if err := json.Unmarshal(state, &history); err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}
	latest := history.Latest()
	if latest == nil {
		return nil, fmt.Errorf("no latest revision found")
	}

	return &State{
		Container: latest.State,
	}, nil
}

type legacyState []*legacyRevision

func (h legacyState) Latest() *legacyRevision {
	if len(h) == 0 {
		return nil
	}
	return h[len(h)-1]
}

func (h legacyState) LatestVersion() int {
	latest := h.Latest()
	if latest == nil {
		return 0
	}
	return latest.Version
}

func (h legacyState) Get(version int) *legacyRevision {
	for _, revision := range h {
		if revision.Version == version {
			return revision
		}
	}
	return nil
}

type legacyRevision struct {
	Version     int       `json:"version"`
	Name        string    `json:"name"`
	Explanation string    `json:"explanation"`
	Output      string    `json:"output,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	State       string    `json:"state"`

	container *dagger.Container `json:"-"`
}
