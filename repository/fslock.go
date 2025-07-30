package repository

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LockType represents different types of operations that can be locked
type LockType string

const (
	// LockTypeRepo - Repository-level operations (fork setup, remote configuration)
	LockTypeRepo LockType = "repo"
	// LockTypeWorktree - Worktree operations (branch creation, worktree initialization)
	LockTypeWorktree LockType = "worktree"
	// LockTypeGitNotes - Git notes operations (state saves, log updates)
	LockTypeGitNotes LockType = "notes"
)

// RepositoryLockManager provides granular process-level locking for repository operations
// to prevent git concurrency issues when multiple container-use instances
// operate on the same repository simultaneously.
type RepositoryLockManager struct {
	repoPath string
	locks    map[LockType]*RepositoryLock
	mu       sync.Mutex
}

// RepositoryLock provides process-level locking for specific operation types
type RepositoryLock struct {
	lockFile string
	fd       *os.File
	mu       sync.Mutex
}

// NewRepositoryLockManager creates a new repository lock manager for the given repository path.
func NewRepositoryLockManager(repoPath string) *RepositoryLockManager {
	return &RepositoryLockManager{
		repoPath: repoPath,
		locks:    make(map[LockType]*RepositoryLock),
	}
}

// GetLock returns a lock for the specified operation type
func (rlm *RepositoryLockManager) GetLock(lockType LockType) *RepositoryLock {
	rlm.mu.Lock()
	defer rlm.mu.Unlock()

	if lock, exists := rlm.locks[lockType]; exists {
		return lock
	}

	// Create a lock file path based on the repository path and lock type
	lockFileName := fmt.Sprintf("container-use-%x-%s.lock", hashString(rlm.repoPath), string(lockType))
	lockDir := filepath.Join(os.TempDir(), "container-use-locks")
	lockFile := filepath.Join(lockDir, lockFileName)

	lock := &RepositoryLock{
		lockFile: lockFile,
	}

	rlm.locks[lockType] = lock
	return lock
}

// WithLock executes a function while holding the specified lock type
func (rlm *RepositoryLockManager) WithLock(ctx context.Context, lockType LockType, fn func() error) error {
	lock := rlm.GetLock(lockType)
	return lock.WithLock(ctx, fn)
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

	for i := range maxRetries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fd, err := os.OpenFile(rl.lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			if os.IsExist(err) {
				// Lock exists, wait and retry with exponential backoff
				exponentialDelay := baseDelay * time.Duration(math.Pow(2, float64(i)))
				delay := time.Duration(math.Min(float64(exponentialDelay), float64(maxDelay)))

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

// Legacy compatibility - NewRepositoryLock creates a general-purpose lock manager
// for backward compatibility. New code should use NewRepositoryLockManager with specific lock types.
func NewRepositoryLock(repoPath string) *RepositoryLockManager {
	return NewRepositoryLockManager(repoPath)
}
