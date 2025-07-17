# Agent Guidelines for gosh

## Build/Test Commands
- Build: `go build`
- Run: `go run main.go`
- Test all: `go test ./...`
- Test single package: `go test ./internal/builtins`, `go test ./internal/input`, etc.
- Test with verbose: `go test -v ./...`
- Format: `go fmt ./...`
- Lint: `golangci-lint run` (if available)

## Project Structure
- Go shell implementation with main.go entry point
- Internal packages: `builtins` (shell commands), `executor` (command execution), `input` (parsing), `history` (command history)
- Module: `github.com/apriljarosz/gosh`
- Go version: 1.24.4
- Uses testify for testing

## Code Style
- Standard Go formatting with `go fmt`
- Package names: lowercase, single word
- Imports: standard library first, then external, then internal (as seen in main.go)
- **Internal imports**: Use full module paths like `github.com/apriljarosz/gosh/internal/builtins` (Go requires the module name + internal path, not relative paths like `./internal/builtins`)
- Error handling: explicit error checking, use `fmt.Fprintf(os.Stderr, ...)` for errors
- Types: PascalCase for exported (Command, Pipeline), camelCase for unexported
- Functions: PascalCase for exported (ReadLine, ParsePipeline), camelCase for private
- Variables: camelCase for local, PascalCase for exported
- Comments: document exported functions and types
- No special rules files found (.cursorrules, copilot-instructions.md)

## Commit Guidelines
- **ALWAYS commit after completing a feature or making significant changes**
- **ALWAYS write comprehensive tests for new features** - maintain high code coverage
- Run tests before committing: `go test ./...`
- Use descriptive commit messages that explain the "why" not just the "what"
- Commit message format: Brief description, then details if needed
- Auto-push is enabled via post-commit hook - commits will be pushed automatically