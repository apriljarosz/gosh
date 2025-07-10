# Agent Guidelines for gosh

## Build/Test Commands
- Build: `go build`
- Run: `go run main.go`
- Test: `go test ./...`
- Test single package: `go test ./internal/packagename`
- Format: `go fmt ./...`
- Lint: `golangci-lint run` (if available)

## Project Structure
- Go shell implementation with main.go entry point
- Internal packages in `internal/` directory (builtins, executor, input)
- Module: `github.com/april/gosh`
- Go version: 1.24.4

## Code Style
- Standard Go formatting with `go fmt`
- Package names: lowercase, single word
- Imports: standard library first, then external, then internal
- Error handling: explicit error checking, use `fmt.Fprintf(os.Stderr, ...)` for errors
- Variables: camelCase for local, PascalCase for exported
- Functions: PascalCase for exported, camelCase for private
- No special rules files found (.cursorrules, copilot-instructions.md)