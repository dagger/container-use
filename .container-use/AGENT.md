This is a development environment for container-use, a CLI tool that provides containerized environments for coding agents.

DEVELOPMENT WORKFLOW:
- Build: Use 'go build -o cu ./cmd/cu' or 'dagger call build --platform=current export --path ./cu'
- Test: Run 'go test ./...' for all tests, 'go test -short ./...' for unit tests only, or 'go test -count=1 -v ./environment' for integration tests
- Format: Always run 'go fmt ./...' before committing
- Dependencies: Run 'go mod download' to install dependencies, 'go mod tidy' to clean up

AVAILABLE TOOLS:
- Go 1.24.x (matches go.mod requirements)
- Docker (for container runtime needed by the tool)
- Dagger v0.18.11 (matches dagger.json)
- Git (for version control)
- Standard build tools

PROJECT STRUCTURE:
- cmd/cu: Main CLI application entry point
- environment/: Core environment management logic
- mcpserver/: MCP (Model Context Protocol) server implementation  
- examples/: Example configurations and usage
- docs/: Documentation and images
- .dagger/: Dagger module configuration

The project uses Dagger for build automation and includes both unit and integration tests. It's designed to work with MCP-compatible agents like Claude Code and Cursor.

NOTE: CGO is disabled by default for better portability and simpler builds. The project builds fine as a static binary.