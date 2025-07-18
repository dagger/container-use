---
title: CLI Reference
description: "Complete reference for all Container Use CLI commands and options."
icon: terminal
---

Container Use provides a comprehensive CLI for managing isolated development environments. All commands follow the pattern:

```bash
container-use {command} [options] [arguments]
```

**Shorthand:** The `cu` command is an alias for `container-use` and can be used interchangeably.

## Global Options

These options can be used with any command:

- `--help`, `-h` - Show help for a command
- `--version` - Show version information
- `--debug` - Enable debug output

## Commands

### `container-use list`

List all environments and their status.

```bash
container-use list
```

**Options:**
- `--no-trunc` - Don't truncate output
- `--quiet`, `-q` - Only show environment IDs

**Output example:**
```
ID              TITLE                     CREATED       UPDATED
frontend-work   React UI Components       5 mins ago    1 min ago
backend-api     FastAPI User Service      3 mins ago    2 mins ago
```

### `container-use log`

View the commit history and commands executed in an environment.

```bash
container-use log {environment-id}
```

**Options:**
- `--patch`, `-p` - Show patch output with diffs

**Example:**
```bash
container-use log fancy-mallard
# Shows full history with commands and file changes

container-use log fancy-mallard --patch
# Shows history with patch diffs
```

### `container-use diff`

Show the code changes made in an environment compared to its base branch.

```bash
container-use diff {environment-id}
```


**Example:**
```bash
container-use diff fancy-mallard
# Shows full diff output
```

### `container-use checkout`

Check out an environment's branch locally to explore in your IDE.

```bash
container-use checkout {environment-id}
```

**Options:**
- `--branch`, `-b` - Specify branch name to checkout

**Example:**
```bash
container-use checkout fancy-mallard
# Switches to branch 'cu-fancy-mallard'
```

### `container-use terminal`

Open an interactive terminal session inside the environment's container.

```bash
container-use terminal {environment-id}
```

**Example:**
```bash
container-use terminal fancy-mallard
# Opens interactive shell in container
```

### `container-use merge`

Merge an environment's work into your current branch, preserving commit history.

```bash
container-use merge {environment-id}
```

**Options:**
- `--delete`, `-d` - Delete environment after successful merge

**Example:**
```bash
git checkout main
container-use merge fancy-mallard
# Merges environment changes into current branch
```

### `container-use apply`

Apply an environment's changes as staged modifications without commits.

```bash
container-use apply {environment-id}
```

**Options:**
- `--delete`, `-d` - Delete environment after successful apply

**Example:**
```bash
git checkout main
container-use apply fancy-mallard
# Stages all changes for you to commit
```

### `container-use delete`

Delete an environment and clean up its resources.

```bash
container-use delete {environment-id}
```

**Options:**
- `--all` - Delete all environments

**Example:**
```bash
container-use delete fancy-mallard
# Deletes the specified environment

container-use delete --all
# Deletes all environments
```

### `container-use watch`

Monitor environment activity in real-time as agents work.

```bash
container-use watch
```

**Example:**
```bash
container-use watch
# Shows live updates from all active environments
```

### `container-use config`

Manage default environment configurations.

```bash
container-use config {subcommand}
```

**Configuration Management:**
- `show [environment-id]` - Display current configuration
- `import {environment-id}` - Import configuration from an environment

**Base Image:**
- `base-image set {image}` - Set default base image
- `base-image get` - Show current base image
- `base-image reset` - Reset to default base image

**Setup Commands:**
- `setup-command add {command}` - Add setup command
- `setup-command remove {command}` - Remove setup command
- `setup-command list` - List setup commands
- `setup-command clear` - Clear all setup commands

**Install Commands:**
- `install-command add {command}` - Add install command
- `install-command remove {command}` - Remove install command
- `install-command list` - List install commands
- `install-command clear` - Clear all install commands

**Environment Variables:**
- `env set {key} {value}` - Set environment variable
- `env unset {key}` - Unset environment variable
- `env list` - List environment variables
- `env clear` - Clear all environment variables

**Secrets:**
- `secret set {key} {value}` - Set secret
- `secret unset {key}` - Unset secret
- `secret list` - List secrets
- `secret clear` - Clear all secrets

**Agent Integration:**
- `agent [agent]` - Configure MCP server for specific agent (claude, goose, cursor, etc.)

**Example:**
```bash
container-use config show
# Shows current configuration

container-use config base-image set python:3.11
# Sets Python 3.11 as default base image

container-use config setup-command add "pip install -r requirements.txt"
# Adds pip install as setup command
```

### `container-use version`

Display Container Use version information.

```bash
container-use version
```

### `container-use stdio`

Start Container Use as an MCP (Model Context Protocol) server for agent integration.

```bash
container-use stdio
```

**Note:** This command is typically used in agent configuration files, not run directly by users.

### `container-use completion`

Generate shell completion scripts.

```bash
container-use completion {shell}
```

**Supported shells:** bash, zsh, fish, powershell

**Example:**
```bash
container-use completion bash > /etc/bash_completion.d/container-use
# Installs bash completion
```


## Environment IDs

Environment IDs are randomly generated two-word identifiers like `fancy-mallard` or `clever-dolphin`. You can use:

- Full ID: `fancy-mallard`
- Partial ID: `fancy` (if unique)
- Branch name: `cu-fancy-mallard`

## Exit Codes

- `0` - Success
- `1` - General error
- `2` - Command syntax error
- `3` - Environment not found
- `4` - Operation cancelled

## Examples

### Basic Workflow

```bash
# 1. List environments
container-use list

# 2. Review changes
container-use diff clever-dolphin

# 3. Check out locally
container-use checkout clever-dolphin

# 4. Accept work
container-use merge clever-dolphin
```

### Debugging Workflow

```bash
# 1. See what agent did
container-use log problematic-env

# 2. Enter container to debug
container-use terminal problematic-env

# 3. Fix issues...

# 4. Apply changes
container-use apply problematic-env
```

### Monitoring Workflow

```bash
# 1. Start watching
container-use watch

# 2. Agent works in background...

# 3. See specific environment
container-use log active-env
```

