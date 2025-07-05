# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go package providing an `slog.Handler` implementation for AWS Lambda functions with advanced logging controls. The package automatically configures log format (JSON/text) and level based on AWS Lambda environment variables.

## Commands

### Testing
- `make test` - Run all tests with race detection
- `go test -v -race ./...` - Run tests directly
- `go test -coverprofile=coverage.txt ./...` - Generate coverage report

### Benchmarking
- `make benchmark` - Run benchmarks for 10 seconds with memory stats

### Building
- `go build ./...` - Build the package

### Code Quality
- `gofmt -s -l .` - Check formatting (must have zero output)
- `nilaway ./...` - Run nil safety analysis
- `gosec ./...` - Run security scanning
- `govulncheck ./...` - Check for vulnerabilities

## Architecture

The core handler is in `handler.go` with these key components:

- **Handler struct** - Main slog handler implementation that outputs to io.Writer
- **Environment-based configuration** - Uses AWS Lambda env vars:
  - `AWS_LAMBDA_LOG_LEVEL` (TRACE/DEBUG/INFO/WARN/ERROR/FATAL)
  - `AWS_LAMBDA_LOG_FORMAT` (json/text)
  - `AWS_LAMBDA_FUNCTION_NAME` and `AWS_LAMBDA_FUNCTION_VERSION`
- **Lambda context integration** - Extracts request ID from lambda context
- **Buffering** - Uses sync.Pool for efficient buffer management
- **Custom log levels** - Maps AWS Lambda levels to slog levels with TRACE and FATAL extensions

The handler outputs structured logs with AWS Lambda metadata including function name, version, and request ID when available.

## Testing Strategy

- Tests are split between external (`handler_test.go`) and internal (`handler_internal_test.go`)
- Example usage in `example_test.go`
- Memory leak testing with `go.uber.org/goleak`
- Race condition testing enabled by default
