package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RepositoryLock provides process-level locking for repository operations
// to prevent git concurrency issues when multiple container-use instances
// operate on the same repository simultaneously.
type RepositoryLock struct {
	lockFile string
	fd       *os.File
	mu       sync.Mutex
}

// NewRepositoryLock creates a new repository lock for the given repository path.
func NewRepositoryLock(repoPath string) *RepositoryLock {
	// Create a lock file path based on the repository path
	// Use a hash or sanitized path to avoid filesystem issues
	lockFileName := fmt.Sprintf("container-use-%x.lock", hashString(repoPath))
	lockDir := filepath.Join(os.TempDir(), "container-use-locks")
	lockFile := filepath.Join(lockDir, lockFileName)

	return &RepositoryLock{
		lockFile: lockFile,
	}
}

// Lock acquires the repository lock with exponential backoff retry.
// This prevents multiple processes from performing conflicting git operations simultaneously.
func (rl *RepositoryLock) Lock(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.fd != nil {
		// Already locked
		return nil
	}

	// Ensure lock directory exists
	if err := os.MkdirAll(filepath.Dir(rl.lockFile), 0755); err != nil {
		return fmt.Errorf("failed to create lock directory: %w", err)
	}

	// Try to acquire lock with exponential backoff
	const maxRetries = 30
	const baseDelay = 50 * time.Millisecond
	const maxDelay = 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fd, err := os.OpenFile(rl.lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			if os.IsExist(err) {
				// Lock exists, wait and retry with exponential backoff
				delay := baseDelay * time.Duration(1<<min(i, 6)) // Cap at 64x base delay
				if delay > maxDelay {
					delay = maxDelay
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
					continue
				}
			}
			return fmt.Errorf("failed to create lock file: %w", err)
		}

		// Write process info to lock file for debugging
		fmt.Fprintf(fd, "pid:%d\ntime:%s\n", os.Getpid(), time.Now().Format(time.RFC3339))

		rl.fd = fd
		return nil
	}

	return fmt.Errorf("failed to acquire repository lock after %d retries", maxRetries)
}

// Unlock releases the repository lock.
func (rl *RepositoryLock) Unlock() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.fd == nil {
		// Not locked
		return nil
	}

	// Close and remove lock file
	rl.fd.Close()
	err := os.Remove(rl.lockFile)
	rl.fd = nil

	return err
}

// WithLock executes a function while holding the repository lock.
func (rl *RepositoryLock) WithLock(ctx context.Context, fn func() error) error {
	if err := rl.Lock(ctx); err != nil {
		return err
	}
	defer rl.Unlock()

	return fn()
}

// hashString creates a simple hash of a string for use in filenames
func hashString(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h = (h ^ uint32(s[i])) * 16777619
	}
	return h
}
