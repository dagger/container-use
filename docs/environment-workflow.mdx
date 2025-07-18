---
title: Environment Workflow
description: "Master the workflow of creating, observing, and managing containerized environments. Learn when to merge work, iterate on prompts, or start fresh."
icon: arrows-rotate
---

## What are Environments?

Each environment in Container Use is an **isolated workspace** that combines:

- **🌳 Git Branch**: Dedicated branch tracking all changes and history
- **📦 Container**: Isolated runtime with your code and dependencies
- **📝 Complete History**: Every command, file change, and container state automatically tracked

Think of environments as **disposable sandboxes** where agents can work safely without affecting your main codebase.

## The Core Workflow

Container Use follows a simple but powerful pattern:

<Steps>
  <Step title="Agent Creates Fresh Environment">
    Every agent session starts with a brand new environment from your current
    branch state.
  </Step>
  <Step title="Agent Works in Isolation">
    The agent makes changes, runs commands, and builds features completely
    isolated from your work.
  </Step>
  <Step title="You Observe the Work">
    Use Container Use commands to see what the agent did without disrupting your
    local setup.
  </Step>
  <Step title="Make a Decision">
    Accept good work using merge or apply, iterate with refined prompts, or
    discard failed attempts.
  </Step>
</Steps>

## Observing Agent Work

You have two modes for inspecting what an agent accomplished:

### Quick Assessment (Non-Interactive)

Perfect for getting the gist and deciding next steps:

```bash
# See all environments at a glance
container-use list

# View exactly what the agent did
container-use log fancy-mallard

# See code changes without checking out
container-use diff fancy-mallard
```

<Card title="When to use" icon="eye">
  Use quick assessment when you want to rapidly understand if the agent is on
  the right track, see what files changed, or review the approach before diving
  deeper.
</Card>

<Note>
  🔒 **Secret Security**: If the agent used any secrets (API keys, database
  credentials), these were resolved within the container environment - agents
  can use your credentials without the AI model ever seeing the actual values.
</Note>

### Deep Exploration (Interactive)

When you need hands-on understanding:

```bash
# Drop into the live container environment
container-use terminal fancy-mallard

# Bring changes into your local workspace/IDE
container-use checkout fancy-mallard
```

<Card title="When to use" icon="magnifying-glass">
  Use deep exploration when you need to test functionality, debug issues,
  understand complex changes, or review code thoroughly in your IDE.
</Card>

## Making Decisions

After observing the agent's work, you have three paths forward:

### ✅ Accept Work

When the agent succeeded and you're happy with the results, you have two options:

**Option 1: Merge (Preserve History)**
```bash
# Merge the environment into your current branch
container-use merge fancy-mallard

# Clean up (optional)
container-use delete fancy-mallard
```

**Option 2: Apply (Customize Commit)**
```bash
# Apply changes as staged modifications
container-use apply fancy-mallard

# Review and commit with your own message
git status
git commit -m "Your custom commit message"

# Clean up (optional)
container-use delete fancy-mallard
```

Choose `merge` to preserve the agent's commit history, or `apply` to create your own commit message and review changes before committing.

### 🔄 Iterate & Refine

When the agent is close but needs refinement:

```bash
# A. Continue the existing chat
# Prompt: "[refined instructions]"
# B. Start a new chat and continue working in the same environment.
# Prompt: "Work in the fancy-mallard environment and [refined instructions]"
```

The agent will resume in the existing environment with all previous work intact. Perfect for:
- Adding missing features
- Fixing bugs the agent introduced
- Adjusting styling or behavior
- Building on partial progress

### 🗑️ Start Fresh

When the agent went down the wrong path:

```bash
# Discard the environment
container-use delete fancy-mallard

# Start over with a new prompt
# The agent will create a fresh environment from your current branch
```

You're back to your last known good state (your current branch) and can try a different approach.

## Resuming Work in Environments

To have a new chat continue work in an existing environment, simply mention the environment ID in your prompt:

<CodeGroup>
```text Good Resume Prompts
"Work in the fancy-mallard environment and add user authentication"

"Continue in environment fancy-mallard - the tests are failing, please fix them"

"Using the fancy-mallard environment, add CSS styling to make it look modern"

````

```text What Happens
The agent will:
✅ Reuse the existing container state
✅ Have access to all previous work
✅ Continue from where it left off
✅ Maintain the same dependencies and setup
````

</CodeGroup>

## Practical Examples

### Example 1: Happy Path Workflow

```bash
# 1. Agent creates environment and builds feature
$ container-use list
ID            TITLE                    CREATED       UPDATED
fancy-mallard Flask App with Login     2 mins ago    30 secs ago

# 2. Quick check - looks good!
$ container-use diff fancy-mallard
+def login():
+    # Authentication logic
+    pass

# 3. Accept the work (choose merge or apply)
$ container-use merge fancy-mallard
Updating abc123..def456
Fast-forward
 app.py | 15 +++++++++++++++
 1 file changed, 15 insertions(+)
```

### Example 2: Iteration Workflow

```bash
# 1. Check what agent built
$ container-use log fancy-mallard
def4567a2  Add basic login form (30 minutes ago)
$ flask run

# 2. Needs improvement - continue in same environment
# Prompt: "Work in fancy-mallard environment and add password validation"

# 3. Check again after agent works
$ container-use diff fancy-mallard
# Now shows both original work + new validation
```

### Example 3: Recovery Workflow

```bash
# 1. Agent's approach isn't working
$ container-use log problematic-env
# Shows agent went down wrong path with complex dependency issues

# 2. Cut losses and start fresh
$ container-use delete problematic-env

# 3. Try different approach
# New prompt: "Create a simple REST API using FastAPI instead of Flask"
# Agent creates fresh environment from clean state
```

## Managing Multiple Environments

You can have multiple agents working simultaneously:

```bash
$ container-use list
ID              TITLE                     CREATED       UPDATED
frontend-work   React UI Components       5 mins ago    1 min ago
backend-api     FastAPI User Service      3 mins ago    2 mins ago
data-pipeline   ETL Processing Script     1 min ago     30 secs ago
```

Each environment is completely isolated - no conflicts, no interference.

## Best Practices

- **Start with Quick Assessment**: Always use `container-use diff` and `container-use log` first. Most of the time, this gives you enough information to decide next steps without the overhead of checking out or entering containers.

- **Merge vs Apply**: Use `merge` when you want to preserve the agent's commit history and understand how the work evolved. Use `apply` when you want to create clean, customized commits or review changes before committing.

- **Don't Be Afraid to Delete**: Environments are disposable by design. If an agent gets stuck or goes down the wrong path, it's often faster to delete and start fresh than to try to fix problematic work.

- **Use Specific Resume Prompts**: When resuming work, be specific about what you want. Instead of "continue working", say "work in ENV-ID and add error handling to the upload function".

- **Keep Your Branch Clean**: Your main working branch should only contain merged, tested work. Use environments for all experimental and agent-driven development.

## Essential Commands Reference

| | |
| --- | --- |
| `container-use list` | See all environments and their status |
| `container-use watch` | Monitor environment activity in real-time as agents work |
| `container-use log <env-id>` | View commit history and commands to understand what the agent did |
| `container-use diff <env-id>` | Quick assessment of code changes |
| `container-use terminal <env-id>` | Enter live container to debug, test, or explore |
| `container-use checkout <env-id>` | Bring changes to local IDE for detailed review |
| `container-use merge <env-id>` | Accept work preserving agent's commit history |
| `container-use apply <env-id>` | Apply as staged changes to customize commits |
| `container-use delete <env-id>` | Discard environment and start over |
| `container-use config` | Configure default settings for new environments |
| `container-use version` | Display version information |

## Next Steps

<CardGroup cols={2}>
  <Card
    title="Environment Configuration"
    icon="gear"
    href="/environment-configuration"
  >
    Configure your project's default environment setup
  </Card>
  <Card
    title="Join Community"
    icon="discord"
    href="https://container-use.com/discord"
  >
    Share your environment workflow strategies in #container-use
  </Card>
</CardGroup>
