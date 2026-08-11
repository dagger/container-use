package environment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAllowedShell(t *testing.T) {
	allowed := []string{
		"sh", "/bin/sh",
		"bash", "/bin/bash",
		"zsh", "/bin/zsh",
		"ash", "/bin/ash",
	}
	for _, shell := range allowed {
		assert.True(t, isAllowedShell(shell), "expected %q to be allowed", shell)
	}

	blocked := []string{
		"/bin/echo",
		"/bin/cat",
		"/bin/rm",
		"python3",
		"../../bin/sh",
		"",
	}
	for _, shell := range blocked {
		assert.False(t, isAllowedShell(shell), "expected %q to be blocked", shell)
	}
}
