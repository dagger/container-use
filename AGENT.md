This is a development environment for container-use, a CLI tool that provides containerized environments for coding agents.

container-use is designed to work with MCP-compatible agents like Claude Code and Cursor.

ENVIRONMENT:

- ALWAYS use ONLY Environments for ANY and ALL file, code, or shell operations—NO EXCEPTIONS—even for simple or generic requests.
- DO NOT install or use the git cli with the environment_run_cmd tool. All environment tools will handle git operations for you. Changing ".git" yourself will compromise the integrity of your environment.
- You MUST inform the user how to view your work using `container-use log <env_id>` AND `container-use checkout <env_id>`. Failure to do this will make your work inaccessible to others.

DEVELOPMENT WORKFLOW:

- Build: Use 'go build -o container-use ./cmd/container-use' or 'dagger call build --platform=current export --path ./container-use'
- Test: Run 'go test ./...' for all tests, 'go test -short ./...' for unit tests only, or 'go test -count=1 -v ./environment' for integration tests
- Format: Always run 'go fmt ./...' before committing
- Lint: Run 'golangci-lint run' to check for linting issues
- Dependencies: Run 'go mod download' to install dependencies, 'go mod tidy' to clean up

MANUAL STDIO TESTING:
- IMPORTANT: container-use cannot be manually tested in its own /workdir. First initialize a new git repository elsewhere on disk (e.g., /tmp/test-repo) and manually test from there
- Test stdio interface: Use 'echo $request | timeout $seconds container-use stdio' where:
  - $request is a JSON-formatted MCP request (e.g., '{"jsonrpc":"2.0","method":"ping","id":1}')
  - $seconds is timeout duration (e.g., 10 for 10 seconds)
  - Example: 'echo '{"jsonrpc":"2.0","method":"ping","id":1}' | timeout 10 container-use stdio'
- For multiline requests, use printf or a here-doc instead of echo
- Use 'jq' to format JSON responses for readability: '... | jq .'
- Common test requests:
  - Ping: '{"jsonrpc":"2.0","method":"ping","id":1}'
  - List tools: '{"jsonrpc":"2.0","method":"tools/list","id":1}'
  - Initialize: '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{}}},"id":1}'

DAGGER MODULE (more details in .dagger/):

- Build: 'dagger call build export --path ./container-use'
- Test: 'dagger call test' or 'dagger call test --integration=false'

PROJECT STRUCTURE:

- cmd/container-use: Main CLI application entry point
- environment/: Core environment management logic
- mcpserver/: MCP (Model Context Protocol) server implementation
- examples/: Example configurations and usage
- docs/: Documentation and images
- .dagger/: Dagger module configuration

DOCS:

- Documentation is in `./docs`, written using Mintlify
- When making changes, make sure the files are properly formatted in mdx
- To start a preview, run `mint dev` from the docs folder