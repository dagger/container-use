package environment

import (
	"fmt"
	"strings"
	"sync"
)

type Notes struct {
	items []string
	mu    sync.Mutex
}

func (n *Notes) Add(format string, a ...any) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.items = append(n.items, fmt.Sprintf(format, a...))
}

func (n *Notes) AddCommand(command string, exitCode int, stdout, stderr string) {
	msg := fmt.Sprintf("$ %s", strings.TrimSpace(command))
	if exitCode != 0 {
		msg += fmt.Sprintf("\nexit %d", exitCode)
	}
	if strings.TrimSpace(stdout) != "" {
		msg += fmt.Sprintf("\n%s", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		msg += fmt.Sprintf("\nstderr: %s", stderr)
	}

	n.Add("%s", msg)
}

func (n *Notes) Clear() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.items = []string{}
}

func (n *Notes) String() string {
	n.mu.Lock()
	defer n.mu.Unlock()

	return strings.TrimSpace(strings.Join(n.items, "\n"))
}

func (n *Notes) Pop() string {
	n.mu.Lock()
	defer n.mu.Unlock()

	out := strings.TrimSpace(strings.Join(n.items, "\n"))
	n.items = []string{}

	return out
}
