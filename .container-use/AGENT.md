This is a development environment for container-use, a CLI tool that provides containerized environments for coding agents.

container-use is designed to work with MCP-compatible agents like Claude Code and Cursor.

DEVELOPMENT WORKFLOW:
- Build: Use 'go build -o cu ./cmd/cu' or 'dagger call build --platform=current export --path ./cu'
- Test: Run 'go test ./...' for all tests, 'go test -short ./...' for unit tests only, or 'go test -count=1 -v ./environment' for integration tests
- Format: Always run 'go fmt ./...' before committing
- Lint: Run 'golangci-lint run' to check for linting issues
- Dependencies: Run 'go mod download' to install dependencies, 'go mod tidy' to clean up

DAGGER MODULE (more details in .dagger/):
- Build: 'dagger call build export --path ./cu'
- Test: 'dagger call test' or 'dagger call test --integration=false'

AVAILABLE TOOLS:
- Go 1.24.x (matches go.mod requirements)
- Docker (for container runtime needed by the tool)
- Dagger v0.18.11 (matches dagger.json)
- Git with test user configured (test dependency, NOT for version control)
- golangci-lint v1.61.0 (Go linter with various checks)

PROJECT STRUCTURE:
- cmd/cu: Main CLI application entry point
- environment/: Core environment management logic
- mcpserver/: MCP (Model Context Protocol) server implementation
- examples/: Example configurations and usage
- docs/: Documentation and images
- .dagger/: Dagger module configuration