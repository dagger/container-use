<div align="center">
  <img src="./docs/images/container-use.png" align="center" alt="Container use: Development environments for coding agents." />
  <h1 align="center">container-use</h2>
  <p align="center">Containerized environments for coding agents. (📦🤖) (📦🤖) (📦🤖)</p>
  <p align="center">
    <img src="https://img.shields.io/badge/stability-experimental-orange.svg" alt="Experimental" />
    <a href="https://opensource.org/licenses/Apache-2.0">
      <img src="https://img.shields.io/badge/License-Apache_2.0-blue.svg">
    </a>
    <a href="https://container-use.com/discord">
      <img src="https://img.shields.io/discord/707636530424053791?logo=discord&logoColor=white&label=Discord&color=7289DA" alt="Discord">
    </a>
    <a href="https://github.com/clinebot/awesome-claude-code">
      <img src="https://awesome.re/mentioned-badge.svg" alt="Mentioned in Awesome Claude Code">
    </a>
  </p>
</div>

**Container Use** lets coding agents do their work in parallel environments without getting in your way. Go from babysitting one agent at a time to enabling multiple agents to work safely and independently with your preferred stack. See the [full documentation](https://container-use.com).

<p align='center'>
    <img src='./docs/images/demo.gif' width='700' alt='container-use demo'>
</p>

It's an open-source MCP server that works as a CLI tool with Claude Code, Cursor, and other MCP-compatible agents. Powered by [Dagger](https://dagger.io).

* 📦 **Isolated Environments**: Each agent gets a fresh container in its own git branch - run multiple agents without conflicts, experiment safely, discard failures instantly.
* 👀 **Real-time Visibility**: See complete command history and logs of what agents actually did, not just what they claim.
* 🚁 **Direct Intervention**: Drop into any agent's terminal to see their state and take control when they get stuck.
* 🎮 **Environment Control**: Standard git workflow - just `git checkout <branch_name>` to review any agent's work.
* 🌎 **Universal Compatibility**: Works with any agent, model, or infrastructure - no vendor lock-in.

---

🦺 This project is in early development and actively evolving. Submit issues and/or reach out to us on [Discord](https://container-use.com/discord) in the #container-use channel.

---

## Quick Start

### Install

```sh
# macOS (recommended)
brew install dagger/tap/container-use

# All platforms
curl -fsSL https://raw.githubusercontent.com/dagger/container-use/main/install.sh | bash
```

### Setup with Your Agent

Container Use works with any MCP-compatible agent. The setup is always the same: **add `container-use stdio` as an MCP server**.

**👉 [Complete setup guide for all agents (Cursor, Goose, VSCode, etc.)](https://container-use.com/quickstart)**

**Example with Claude Code:**

```sh
# Add Container Use MCP server
cd /path/to/repository
claude mcp add container-use -- container-use stdio

# Add agent rules (optional)
curl https://raw.githubusercontent.com/dagger/container-use/main/rules/agent.md >> CLAUDE.md
```

<details>
<summary>💡 Command Shortcut</summary>

The `container-use` command is also available as `cu` for convenience. Both commands work identically:
- `container-use stdio` (used in documentation)
- `cu stdio` (shortcut)

</details>

### Try It

Ask your agent to create something:
> Create a hello world app in python using flask

Your agent will work in an isolated environment and give you URLs to view the app and explore the code!