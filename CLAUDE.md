# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go `slog.Handler` for AWS Lambda functions that auto-configures log format (JSON/text) and level from Lambda environment variables. Import alias is `sloglambda`.

## Commands

### Testing
- `make test` - Run all tests with race detection
- `go test -v -race -run TestHandler/slogtest ./...` - Run a specific test
- `go test -coverprofile=coverage.txt ./...` - Generate coverage report

### Benchmarking
- `make benchmark` - Run benchmarks for 10 seconds with memory stats

### Code Quality (all enforced in CI)
- `gofmt -s -l .` - Check formatting (must have zero output)
- `nilaway ./...` - Nil safety analysis
- `gosec ./...` - Security scanning
- `govulncheck ./...` - Vulnerability checking

## Architecture

Single-file handler in `handler.go`. The `Handler` struct implements `slog.Handler` and writes to an `io.Writer`.

Key design decisions:
- **Functional options pattern** - `Option` type (`func(*Handler)`) for configuration: `WithJSON()`, `WithText()`, `WithLevel()`, `WithSource()`, `WithType()`, `WithoutTime()`
- **Environment auto-detection** - `NewHandler` reads `AWS_LAMBDA_LOG_LEVEL` and `AWS_LAMBDA_LOG_FORMAT` at construction time; options override env values
- **Custom level mapping** - TRACE = `slog.LevelDebug - 4`, FATAL = `slog.LevelError + 4`; `lambdaLoggerLevelString` renders these back for output
- **`logRecord` (map[string]any)** - Internal representation built in `Handle()`, supports nested groups; cleaned of empty sub-records before serialization
- **Value normalization** - `normalizeValue` resolves `slog.LogValuer` and converts typed values; `normalizeAnyValue` handles `error` and `json.Marshaler`
- **Buffer pooling** - `sync.Pool` with 1KB initial alloc, 16KB max retention
- **Lambda metadata** - Injects function name, version from env, and request ID from `lambdacontext` into a `record` group

## Testing

- External tests (`handler_test.go`) use `slogtest.Run` for `slog.Handler` conformance - both JSON and text formats
- Internal tests (`handler_internal_test.go`) cover unexported helpers: level parsing, level string rendering, `logRecord` operations, text serialization, value normalization
- `main_test.go` sets Lambda env vars and runs `goleak.VerifyTestMain` for goroutine leak detection
- `value_test.go` covers `slog.LogValuer` resolution and complex types like `http.Request`

## CI

GitHub Actions runs on push to main and PRs: test (race, multi-OS), build, coverage, gosec, govulncheck, gofmt, nilaway.
