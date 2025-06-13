# Environments

An **environment** is an isolated, containerized development workspace that combines Docker containers with Git branches to provide agents with safe, persistent workspaces.

## What is an Environment?

Each environment consists of:
- **Git Branch**: Dedicated branch tracking all changes and history
- **Container**: Dagger container with your code and dependencies
- **History**: Versioned snapshots of container state changes appended to the branch as notes
- **Configuration**: Base image, setup commands, secrets, and instructions that can be checked into the source repo.

## Key Features

- **Branch-Based**: Each environment is a Git branch that syncs into the container-use/ remote
- **Isolation**: Each environment runs in its own container and branch
- **Persistence**: All changes automatically committed with full history
- **Standard Git**:
  - Use `git log` to view source code history
  - Use `git log --notes=container-use` to view container state history
  - Use `git checkout env-branch` to inspect any environment's work - each env branch tracks the upstream container-use/
- **State Recovery**: Container states stored in Git notes for reconstruction

## How It Works

When you create an environment, container-use:

1. **Creates a new Git branch** in your source repo (e.g., `env-name/adverb-animal`)
2. **Sets up a container-use remote branch** inside `~/.config/container-use/repos/project/`
3. **Sets up a worktree copy of the branch** in `~/.config/container-use/worktrees/project/`
4. **Spins up a Dagger container** with that worktree copied into `/workdir`

When an agent runs commands:

1. **Commands execute** inside the isolated container
2. **File changes get written** back to the container filesystem
3. **Everything gets committed** to the environment's Git branch automatically
4. **Container state snapshots** are stored as Git notes for later recovery

Each environment is just a Git branch that your source repo tracks on the container-use/ remote. You can inspect any environment's work using standard Git commands, and the container state can always be reconstructed from an environment branch's Git history.

## Architecture

```
projectName/ Source Repo            container-use/ Remote
├── main                   ←──→ ├── main
├── feature-branch         ←──→ ├── feature-branch
└── env-name/adverb-animal ←──→ └── env-name/adverb-animal
                                       │
                                       │ (host filesystem implementation)
                                       ▼
                    ~/.config/container-use/
                    ├── repos/projectName/ (bare)
                    └── worktrees/env-name/adverb-animal (only env branches become worktrees)
                        ├── .git -> ../../repos/projectName/worktrees/env-name/adverb-animal
                        └── (your code)
                            │
                            ▼
                        Container
                        └── /workdir
```

The diagram shows how branches sync between your source repo and the container-use remote. Each environment branch (like `env-name/adverb-animal`) exists in both places and stays synchronized.

Below the branch level, the system creates a bare Git repository and worktree in `~/.config/container-use/` - this is plumbing to make the Git operations work with minimal modifications to your source repository. The worktree contains a copy of your code that gets mounted into the Docker container at `/workdir`.

So the flow is: **Branch** (the logical environment) → **Worktree** (filesystem implementation) → **Container** (where code actually runs).

## Files

- `environment.go` - Core environment management
- `git.go` - Worktree and Git integration
- `filesystem.go` - File operations within containers

## Testing

The environment package includes comprehensive unit and integration tests that verify critical user behaviors:

### Running Tests

#### Using Go directly:
```bash
# Run only unit tests
go test ./environment -short

# Run all tests including integration tests (requires Dagger)
go test ./environment

# Run specific test suites
go test ./environment -run TestWorkPersistsAcrossSessions

# Run with verbose output
go test ./environment -v
```

#### Using Dagger (recommended for CI):
```bash
# Run unit tests only
dagger call test-environment

# Run all tests including integration tests
dagger call test-environment --integration

# Run with verbose output
dagger call test-environment --verbose --integration

# Run all tests in the project
dagger call test --pkg ./...
```

### Test Coverage

**Unit Tests** (`-short` mode):
- Git command error handling
- Worktree path generation
- Empty directory handling
- Selective file staging (Python cache, binaries)

**Integration Tests** (full mode):
- Work persistence across sessions
- Automatic change tracking
- Environment isolation
- Edge case handling (Python cache, binary files, large files)
- Performance with 100+ file projects
- Configuration persistence

### Known Issues

The test suite documents several known bugs with skipped tests:
- Python `__pycache__` directories cause git operations to fail
- Directories with only binary files aren't handled correctly
- Environment variables don't persist across `Update()` calls
- **Worktree caching bug**: After `Update()`, the container uses the cached initial worktree state instead of the latest changes (dag.Host().Directory() caching issue)
- **Upload caching**: The `Upload()` function may upload stale versions of files due to the same caching issue

These failing tests are intentionally included to document expected behavior once the bugs are fixed.
