---
title: Quickstart
description: "Get started with Container Use in 5 minutes."
icon: rocket
---

In this quickstart, you'll install Container Use, connect it to your coding agent, and experience the core workflow: agents work in isolated environments while your files stay untouched, then review and decide what to do with their work.

## 1. Install Container Use

Make sure you have [Docker](https://www.docker.com/get-started) and Git installed before starting.

<Tabs>
  <Tab title="Homebrew (macOS)">

  ```sh
  brew install dagger/tap/container-use
  container-use version   # → confirms install
  ```

  </Tab>

  <Tab title="Shell Script (All Platforms)">

  ```sh
  curl -fsSL https://raw.githubusercontent.com/dagger/container-use/main/install.sh | bash
  container-use version
  ```

  </Tab>

  <Tab title="Build from Source">

  ```sh
  git clone https://github.com/dagger/container-use.git
  cd container-use
  go build -o container-use ./cmd/container-use
  sudo mv container-use /usr/local/bin/
  container-use version
  ```

  </Tab>
</Tabs>

## 2. Point your agent at Container Use

Container Use works with any MCP-compatible agent: Just add `container-use stdio` as an MCP server. This example uses Claude Code but you can view [instructions for other agents](/agent-integrations).

<Steps>
  <Step title="Add MCP Configuration">
    ```sh
    cd /path/to/repository
    claude mcp add container-use -- container-use stdio
    ```
  </Step>

  <Step title="Add Agent Rules (Optional)">
    Save CLAUDE.md file at the root of your repository:

    ```sh
    curl https://raw.githubusercontent.com/dagger/container-use/main/rules/agent.md >> CLAUDE.md
    ```
  </Step>

  <Step title="Trust Only Container Use Tools (Optional)">
    For maximum security, restrict Claude Code to only use Container Use tools:

    ```sh
    claude --allowedTools mcp__container-use__environment_checkpoint,mcp__container-use__environment_create,mcp__container-use__environment_add_service,mcp__container-use__environment_file_delete,mcp__container-use__environment_file_list,mcp__container-use__environment_file_read,mcp__container-use__environment_file_write,mcp__container-use__environment_open,mcp__container-use__environment_run_cmd,mcp__container-use__environment_update
    ```
  </Step>
</Steps>

## 3. Run your first parallel task

Let's create a demo repository and ask your agent to build something:

```sh
# start a demo repo
mkdir hello
cd hello
git init
touch README.md
git add README.md
git commit -m "initial commit"
```

Now prompt your agent to do something:
```text
"Create a Flask hello‑world app in Python."
```

After a short run you'll see something like:

```text
✅ App running at http://127.0.0.1:58455
🔍 View files:  container-use checkout {id}
📋 Dev log:     container-use log {id}
```

<Note>
Replace `{id}` with your actual environment ID like `fancy-mallard`
</Note>

Notice your local directory is still empty, because the agent worked in an isolated environment:

```sh
$ ls
README.md
```

You can see all environments with `container-use list`.

## 4. Review the work

See what the agent changed:
```sh
container-use diff {id}
```

Check out the environment locally to explore:
```sh
container-use checkout {id}
```

## 5. Accept or discard

Accept the work and keep the agent's commit history:
```sh
container-use merge {id}
```

Or stage the changes to create your own commit:
```sh
container-use apply {id}
```

🎉 That's it – you've run an agent in parallel, checked its work, and decided what to do with it.
