# Contributing to Container Use

Thank you for your interest in contributing to Container Use! This document provides guidelines and information for contributors.

## Development Setup

1. Ensure you have Go 1.21+ installed
2. Clone the repository
3. Install dependencies: `go mod download`

## Testing

Container Use has both unit and integration tests to ensure code quality and functionality.

### Running Tests

To run all tests:
```bash
go test ./...
```

To run only unit tests (faster, no containers):
```bash
go test -short ./...
```

To run integration tests only:
```bash
go test -run Integration ./environment
```

### Test Structure

The test suite is organized as follows:
- `environment_test.go` - Unit tests for the environment package
- `integration_test.go` - Integration tests that create real containers
- `test_helpers.go` - Shared test utilities and helpers

Integration tests require Docker to be running and will create temporary containers and git repositories.

### Writing Tests

When adding new functionality:
1. Write unit tests for core logic
2. Add integration tests for end-to-end scenarios
3. Use the provided test helpers for common operations
4. Follow the existing test patterns for consistency

## Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Ensure all tests pass before submitting PR

## Submitting Changes

1. Fork the repository
2. Create a feature branch
3. Make your changes with appropriate tests
4. Submit a pull request with clear description

## Questions?

Join our [Discord](https://discord.gg/Nf42dydvrX) in the #container-use channel for discussions and support.