# gosh - Go Shell

A modern shell implementation written in Go, featuring essential shell functionality with a clean, modular architecture.

## Features

### Core Functionality
- **Interactive REPL** with command prompt
- **Built-in commands**: `cd`, `pwd`, `exit`, `help`, `env`, `history`, `jobs`, `fg`, `bg`
- **External command execution** with full PATH support
- **Command history** with persistent storage in `~/.gosh_history`
- **Tab completion** for commands and file paths

### I/O Redirection
- **Output redirection**: `command > file.txt`
- **Append redirection**: `command >> file.txt`
- **Input redirection**: `command < file.txt`

### Advanced Features
- **Pipes**: Chain commands with `|` (supports multiple pipes)
- **Background jobs**: Run commands with `&`
- **Job control**: Manage background jobs with `jobs`, `fg`, `bg`
- **Environment variables**: Full support with `$VAR` and `${VAR}` expansion
- **Command parsing** with proper tokenization
- **Optional advanced line editing**: Arrow key navigation and history browsing

## Installation

### Prerequisites
- Go 1.24.4 or later

### Build from source
```bash
git clone https://github.com/apriljarosz/gosh.git
cd gosh
go build
./gosh
```

### Install globally
```bash
go install github.com/apriljarosz/gosh@latest
```

## Usage

### Basic Commands
```bash
gosh> pwd
/Users/april/repos/personal/gosh

gosh> cd /tmp
gosh> pwd
/tmp

gosh> help
gosh - Go Shell
Built-in commands:
  cd [dir]  - Change directory
  pwd       - Print working directory
  help      - Show this help
  exit      - Exit the shell
```

### I/O Redirection
```bash
# Output redirection
gosh> echo "Hello World" > hello.txt
gosh> cat hello.txt
Hello World

# Append to file
gosh> echo "Second line" >> hello.txt

# Input redirection
gosh> wc -l < hello.txt
       2
```

### Pipes
```bash
# Single pipe
gosh> ls | wc -l
       8

# Multiple pipes
gosh> ls | head -5 | wc -l
       5

# Pipe with redirection
gosh> ps aux | grep go | wc -l > process_count.txt
```

### Background Jobs
```bash
# Run command in background
gosh> sleep 10 &
[1] 12345

# Continue using shell while command runs
gosh> echo "Shell is still responsive"
Shell is still responsive

# Manage jobs
gosh> jobs
[1]  Running    sleep 10 &

# Bring job to foreground
gosh> fg 1
```

### Advanced Line Editing (Optional)
By default, gosh uses simple line input for maximum compatibility. For users who want advanced features like arrow key navigation and history browsing, enable advanced mode:

```bash
# Enable advanced line editing
export GOSH_ADVANCED_EDITING=1
./gosh

# Now you can use:
# - Up/Down arrows: Browse command history
# - Left/Right arrows: Move cursor within line
# - Home/End: Jump to beginning/end of line
# - Tab: Smart completion with common prefix
# - Ctrl+C: Cancel current line
```

**Note**: Advanced line editing uses raw terminal mode which can sometimes cause display issues on certain terminals. The simple mode (default) is more reliable and matches the behavior of the original mkouhei/gosh implementation.

## Architecture

The shell is built with a modular architecture:

```
gosh/
├── main.go                    # Main REPL loop
├── internal/
│   ├── input/                 # Command parsing and input handling
│   │   └── input.go
│   ├── executor/              # Command execution and I/O redirection
│   │   └── executor.go
│   └── builtins/              # Built-in command implementations
│       └── builtins.go
└── go.mod
```

### Key Components

- **Input Parser**: Tokenizes command lines and handles redirection operators
- **Pipeline Parser**: Supports complex command chains with pipes
- **Command Executor**: Manages process execution with proper I/O handling
- **Built-in Commands**: Implements shell-specific commands that can't be external

## Examples

### Complex Command Combinations
```bash
# Find Go files and count lines
gosh> find . -name "*.go" | xargs wc -l | tail -1 > line_count.txt

# Process monitoring
gosh> ps aux | grep -v grep | grep go > go_processes.txt &

# File operations with pipes
gosh> ls -la | grep "\.go$" | awk '{print $9}' | sort
```

## Roadmap

### Completed ✅
- [x] Environment variable expansion (`$VAR`, `${VAR}`)
- [x] Environment variable management (`env` command)
- [x] Command history with persistent storage
- [x] Tab completion for files and commands
- [x] Signal handling (Ctrl+C)
- [x] Job control (`jobs`, `fg`, `bg` commands)
- [x] Arrow key navigation (optional advanced mode)

### High Priority
- [ ] Better command parsing (quotes, escaping)
- [ ] Globbing support (`*.txt`, `*.go`)
- [ ] Command substitution (`$(command)`)

### Medium Priority
- [ ] Aliases and configuration files
- [ ] More robust signal handling (Ctrl+Z, job suspension)
- [ ] Auto-suggestions based on history
- [ ] Multi-line command support

### Low Priority
- [ ] Scripting support (conditionals, loops)
- [ ] Syntax highlighting
- [ ] Plugin system

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

### Development Setup
```bash
git clone https://github.com/apriljarosz/gosh.git
cd gosh
go mod tidy
go build
go test ./...
```

### Running Tests
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./internal/input
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Built with Go's excellent standard library, particularly:
- `os/exec` for process management
- `bufio` for input handling
- `strings` for command parsing
