package agent

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSupportedAgents(t *testing.T) {
	supported := getSupportedAgents()
	assert.NotEmpty(t, supported)

	// Platform filtering: codex and amazonq are unsupported on native Windows
	for _, a := range supported {
		if runtime.GOOS == "windows" {
			assert.NotContains(t, []string{"codex", "amazonq"}, a.Key)
		}
	}

	// Every known agent must be selectable
	for _, a := range supported {
		_, err := selectAgent(a.Key)
		assert.NoError(t, err, "supported agent %q must be selectable", a.Key)
	}
}

func TestGetSupportedAgentsInstalledFirst(t *testing.T) {
	supported := getSupportedAgents()

	// Property: no installed agent may appear after a non-installed one
	seenNonInstalled := false
	for _, a := range supported {
		installed := agentInstalled(a.Key)
		if seenNonInstalled {
			assert.False(t, installed, "installed agent %q must not appear after a non-installed agent", a.Key)
		}
		if !installed {
			seenNonInstalled = true
		}
	}
}

func TestAgentInstalledUnknownKey(t *testing.T) {
	assert.False(t, agentInstalled("definitely-not-an-agent"))
}
