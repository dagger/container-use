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
  - Use `git log --notes=container-use` to view container operation history
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
3. **Container state is preserved** in the Dagger container's LLB definition
4. **Everything gets committed** to the environment's Git branch automatically
5. **Container state snapshots** are stored as Git notes using `container-use-state` ref
6. **Operation logs** are stored as Git notes using `container-use` ref

Each environment is just a Git branch that your source repo tracks on the container-use/ remote. You can inspect any environment's work using standard Git commands, and the container state can always be reconstructed from an environment branch's Git history and notes.

## Architecture

```
projectName/ Source Repo                container-use/ Remote
├── feature-branch ←──── cu merge/apply ────────┐
├── main (current) ── environment_create ──→ adverb-animal
└── cu-adverb-animal ←──── cu checkout ───────────┘
                                       │
                                       │ (host filesystem implementation)
                                       ▼
                    ~/.config/container-use/
                    ├── repos/projectName/ (bare)
                    └── worktrees/adverb-animal (only env branches become worktrees)
                        ├── .git -> ../../repos/projectName/worktrees/adverb-animal
                        └── (your code)
                            │
                            ▼
                        Container
                        └── /workdir
```

The diagram shows how environment branches sync between your source repo and the container-use remote. When you create an environment, the current branch content gets pushed to the container-use remote as `adverb-animal`. When you checkout an environment, it creates a local tracking branch `cu-adverb-animal` that tracks the remote environment branch. Regular branches like `main` and `feature-branch` stay only in your source repo.

You can accept the environment's work into your current branch using either:
- **`cu merge`** - Preserves the agent's commit history
- **`cu apply`** - Stages changes for you to commit with your own message

Below the branch level, the system creates a bare Git repository and worktree in `~/.config/container-use/` - this is plumbing to make the Git operations work with minimal modifications to your source repository. The worktree contains a copy of your code that gets mounted into the Docker container at `/workdir`.

So the flow is: **Branch** (the logical environment) → **Worktree** (filesystem implementation) → **Container** (where code actually runs).

## Files

- `environment.go` - Core environment management and container operations
- `config.go` - Configuration management and persistence
- `state.go` - Container state serialization and legacy migration
- `service.go` - Service management for multi-container environments
- `note.go` - Git notes management for operation logging
- `filesystem.go` - File operations within containers
- `../repository/git.go` - Worktree and Git integration
- `../repository/repository.go` - High-level repository operations