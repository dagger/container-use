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

func (n *Notes) Clear() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.items = []string{}
}

func (n *Notes) String() string {
	n.mu.Lock()
	defer n.mu.Unlock()

	return strings.Join(n.items, "\n")
}

func (n *Notes) Pop() string {
	n.mu.Lock()
	defer n.mu.Unlock()

	out := strings.Join(n.items, "\n")
	n.items = []string{}

	return out
}
