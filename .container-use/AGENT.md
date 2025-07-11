This is a development environment for container-use, a CLI tool that provides containerized environments for coding agents.

container-use is designed to work with MCP-compatible agents like Claude Code and Cursor.

DEVELOPMENT WORKFLOW:
- Build: Use 'go build -o container-use ./cmd/container-use' or 'dagger call build --platform=current export --path ./container-use'
- Test: Run 'go test ./...' for all tests, 'go test -short ./...' for unit tests only, or 'go test -count=1 -v ./environment' for integration tests
- Format: Always run 'go fmt ./...' before committing
- Lint: Run 'golangci-lint run' to check for linting issues
- Dependencies: Run 'go mod download' to install dependencies, 'go mod tidy' to clean up

MANUAL STDIO TESTING:
- Test stdio interface: Use 'echo $request | timeout $seconds container-use stdio' where:
  - $request is a JSON-formatted MCP request (e.g., '{"jsonrpc":"2.0","method":"ping","id":1}')
  - $seconds is timeout duration (e.g., 10 for 10 seconds)
  - Example: 'echo '{"jsonrpc":"2.0","method":"ping","id":1}' | timeout 10 container-use stdio'
- For multiline requests, use printf or a here-doc instead of echo
- Use 'jq' to format JSON responses for readability: '... | jq .'
- Common test requests:
  - Ping: '{"jsonrpc":"2.0","method":"ping","id":1}'
  - List tools: '{"jsonrpc":"2.0","method":"tools/list","id":1}'
  - Initialize: '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"roots":{"listChanged":true},"sampling":{}}},"id":1}'

DAGGER MODULE (more details in .dagger/):
- Build: 'dagger call build export --path ./container-use'
- Test: 'dagger call test' or 'dagger call test --integration=false'

AVAILABLE TOOLS:
- Go 1.24.5 (matches go.mod requirements)
- Docker v28.3.2 (for container runtime needed by the tool)
- Dagger v0.18.11 (matches dagger.json)
- Git v2.30.2 with test user configured (test dependency, NOT for version control)
- golangci-lint v1.61.0 (Go linter with various checks)
- jq v1.6 (JSON processor for formatting stdio test responses)

PROJECT STRUCTURE:
- cmd/container-use: Main CLI application entry point
- environment/: Core environment management logic
- mcpserver/: MCP (Model Context Protocol) server implementation
- examples/: Example configurations and usage
- docs/: Documentation and images
- .dagger/: Dagger module configuration