package repository

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

// LockType represents different types of operations that can be locked
type LockType string

const (
	// LockTypeUserRepo - Git operations on the environment_source repo (changing remotes, fetching)
	LockTypeUserRepo LockType = "user-repo"
	// LockTypeForkRepo - Git operations on our fork (~/.config/container-use/repos/$repo)
	// Used for committing, creating worktrees, uploading from worktrees (should be rlock for uploads)
	LockTypeForkRepo LockType = "fork-repo"
	// LockTypeNotes - Subset of fork repo operations for saving state, notes etc
	// Notes are a global ref to that repository and we do many operations against them
	LockTypeNotes LockType = "notes"
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
	flock *flock.Flock
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

	lockFileName := fmt.Sprintf("container-use-%x-%s.lock", hashString(rlm.repoPath), string(lockType))
	lockDir := filepath.Join(os.TempDir(), "container-use-locks")
	lockFile := filepath.Join(lockDir, lockFileName)

	err := os.MkdirAll(lockDir, 0755)
	if err != nil {
		slog.Error("Failed to create lock directory", "error", err)
	}

	lock := &RepositoryLock{
		flock: flock.New(lockFile),
	}

	rlm.locks[lockType] = lock
	return lock
}

// WithLock executes a function while holding an exclusive lock for the specified lock type
func (rlm *RepositoryLockManager) WithLock(ctx context.Context, lockType LockType, fn func() error) error {
	return rlm.GetLock(lockType).WithLock(ctx, fn)
}

// WithRLock executes a function while holding a shared (read) lock for the specified lock type.
// Multiple readers can hold the lock simultaneously, but writers will block until all readers release.
func (rlm *RepositoryLockManager) WithRLock(ctx context.Context, lockType LockType, fn func() error) error {
	return rlm.GetLock(lockType).WithRLock(ctx, fn)
}

// Lock acquires an exclusive repository lock.
func (rl *RepositoryLock) Lock(ctx context.Context) error {
	const retryDelay = 100 * time.Millisecond

	locked, err := rl.flock.TryLockContext(ctx, retryDelay)
	if err != nil {
		return fmt.Errorf("failed to acquire exclusive lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("failed to acquire exclusive lock within context timeout")
	}

	return nil
}

// RLock acquires a shared repository lock.
// Multiple processes can hold shared locks simultaneously.
func (rl *RepositoryLock) RLock(ctx context.Context) error {
	const retryDelay = 100 * time.Millisecond

	locked, err := rl.flock.TryRLockContext(ctx, retryDelay)
	if err != nil {
		return fmt.Errorf("failed to acquire shared lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("failed to acquire shared lock within context timeout")
	}

	return nil
}

// Unlock releases the repository lock.
func (rl *RepositoryLock) Unlock() error {
	return rl.flock.Unlock()
}

// WithLock executes a function while holding an exclusive lock.
func (rl *RepositoryLock) WithLock(ctx context.Context, fn func() error) error {
	if err := rl.Lock(ctx); err != nil {
		return err
	}
	defer rl.Unlock()

	return fn()
}

// WithRLock executes a function while holding a shared lock.
func (rl *RepositoryLock) WithRLock(ctx context.Context, fn func() error) error {
	if err := rl.RLock(ctx); err != nil {
		return err
	}
	defer rl.Unlock()

	return fn()
}

// hashString creates a simple hash of a string for use in filenames
func hashString(s string) uint32 {
	h := uint32(2166136261) // FNV-1a 32-bit offset basis
	for i := 0; i < len(s); i++ {
		h = (h ^ uint32(s[i])) * 16777619 // FNV-1a 32-bit prime
	}
	return h
}
