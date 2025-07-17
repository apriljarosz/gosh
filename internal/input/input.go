package input

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"unsafe"

	"github.com/apriljarosz/gosh/internal/history"
)

// Command represents a parsed command with potential redirection
type Command struct {
	Args         []string
	InputFile    string
	OutputFile   string
	AppendOutput bool
	Background   bool
}

// Pipeline represents a series of commands connected by pipes
type Pipeline struct {
	Commands   []*Command
	Background bool
}

// Terminal control structures for raw mode
type termios struct {
	Iflag  uint32
	Oflag  uint32
	Cflag  uint32
	Lflag  uint32
	Cc     [20]uint8
	Ispeed uint32
	Ospeed uint32
}

// LineEditor handles interactive line editing with arrow key support
type LineEditor struct {
	history          *history.History
	originalTty      termios
	rawMode          bool
	completionEngine *CompletionEngine
}

// CompletionEngine handles tab completion for commands and paths
type CompletionEngine struct {
	builtinCommands []string
}

// NewCompletionEngine creates a new completion engine
func NewCompletionEngine() *CompletionEngine {
	return &CompletionEngine{
		builtinCommands: []string{"cd", "pwd", "exit", "help", "env", "history"},
	}
}

// Complete returns possible completions for the given input
func (ce *CompletionEngine) Complete(line string, cursor int) []string {
	if cursor > len(line) {
		cursor = len(line)
	}

	// Find the word being completed
	wordStart := cursor
	for wordStart > 0 && line[wordStart-1] != ' ' {
		wordStart--
	}

	prefix := line[wordStart:cursor]
	words := strings.Fields(line[:cursor])

	// If this is the first word, complete commands
	if len(words) <= 1 {
		return ce.completeCommand(prefix)
	}

	// Otherwise, complete file paths
	return ce.completePath(prefix)
}

// completeCommand completes built-in commands and executables in PATH
func (ce *CompletionEngine) completeCommand(prefix string) []string {
	seen := make(map[string]bool)
	var matches []string

	// Check built-in commands first (they take priority)
	for _, cmd := range ce.builtinCommands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
			seen[cmd] = true
		}
	}

	// Check executables in PATH, avoiding duplicates
	pathMatches := ce.completeExecutables(prefix)
	for _, cmd := range pathMatches {
		if !seen[cmd] {
			matches = append(matches, cmd)
			seen[cmd] = true
		}
	}

	sort.Strings(matches)
	return matches
}

// completeExecutables finds executables in PATH that match the prefix
func (ce *CompletionEngine) completeExecutables(prefix string) []string {
	var matches []string
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return matches
	}

	paths := strings.Split(pathEnv, ":")
	seen := make(map[string]bool)

	for _, dir := range paths {
		if dir == "" {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, prefix) && !seen[name] {
				// Check if it's executable
				if info, err := entry.Info(); err == nil && info.Mode()&0111 != 0 {
					matches = append(matches, name)
					seen[name] = true
				}
			}
		}
	}

	return matches
}

// completePath completes file and directory paths
func (ce *CompletionEngine) completePath(prefix string) []string {
	var matches []string

	// Handle absolute vs relative paths
	dir := "."
	pattern := prefix

	if strings.Contains(prefix, "/") {
		dir = filepath.Dir(prefix)
		pattern = filepath.Base(prefix)
		if dir == "" {
			dir = "/"
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return matches
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, pattern) {
			fullPath := name
			if dir != "." {
				fullPath = filepath.Join(dir, name)
			}

			// Add trailing slash for directories
			if entry.IsDir() {
				fullPath += "/"
			}

			matches = append(matches, fullPath)
		}
	}

	sort.Strings(matches)
	return matches
}

// findCommonPrefix finds the longest common prefix among a list of strings
func findCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	prefix := strs[0]
	for _, s := range strs[1:] {
		for len(prefix) > 0 && !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
		if prefix == "" {
			break
		}
	}
	return prefix
}

// NewLineEditor creates a new line editor with history support
func NewLineEditor(hist *history.History) *LineEditor {
	return &LineEditor{
		history:          hist,
		completionEngine: NewCompletionEngine(),
	}
}

// enableRawMode puts the terminal in raw mode for character-by-character input
func (le *LineEditor) enableRawMode() error {
	fd := int(os.Stdin.Fd())

	// Get current terminal settings
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCGETA, uintptr(unsafe.Pointer(&le.originalTty)))
	if errno != 0 {
		return errno
	}

	// Create raw mode settings - more conservative approach
	raw := le.originalTty
	// Only disable what we absolutely need for raw input
	raw.Lflag &^= syscall.ECHO | syscall.ICANON
	// Keep most other processing enabled to avoid display issues
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	// Apply raw mode settings
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCSETA, uintptr(unsafe.Pointer(&raw)))
	if errno != 0 {
		return errno
	}

	le.rawMode = true
	return nil
}

// disableRawMode restores the original terminal settings
func (le *LineEditor) disableRawMode() error {
	if !le.rawMode {
		return nil
	}

	fd := int(os.Stdin.Fd())
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCSETA, uintptr(unsafe.Pointer(&le.originalTty)))
	if errno != 0 {
		return errno
	}

	le.rawMode = false
	return nil
}

// ReadLineWithArrows reads a line with arrow key support and history navigation
func (le *LineEditor) ReadLineWithArrows() (string, error) {
	fmt.Print("gosh> ")

	if err := le.enableRawMode(); err != nil {
		// Fallback to simple mode if raw mode fails
		return le.readLineSimple()
	}
	defer le.disableRawMode()

	var line []rune
	cursor := 0
	historyPos := le.history.Size()
	originalLine := ""

	for {
		var buf [1]byte
		n, err := os.Stdin.Read(buf[:])
		if err != nil || n == 0 {
			continue
		}

		ch := buf[0]

		switch ch {
		case '\r': // Enter key (in raw mode, Enter sends \r)
			// Move to next line without carriage return
			fmt.Print("\n")
			result := string(line)
			if result != "" {
				le.history.Reset()
			}
			return result, nil

		case '\n': // Handle \n as well just in case
			fmt.Print("\n")
			result := string(line)
			if result != "" {
				le.history.Reset()
			}
			return result, nil

		case '\x03': // Ctrl+C
			fmt.Print("^C\n")
			le.history.Reset()
			return "", fmt.Errorf("interrupted")

		case '\x7f', '\b': // Backspace
			if cursor > 0 {
				line = append(line[:cursor-1], line[cursor:]...)
				cursor--
				le.redrawLine(line, cursor)
			}

		case '\t': // Tab completion
			completions := le.completionEngine.Complete(string(line), cursor)
			if len(completions) == 1 {
				// Single completion - insert it
				completion := completions[0]

				// Replace the prefix with the completion
				wordStart := cursor
				for wordStart > 0 && line[wordStart-1] != ' ' {
					wordStart--
				}

				newLine := append(line[:wordStart], []rune(completion)...)
				newLine = append(newLine, line[cursor:]...)
				line = newLine
				cursor = wordStart + len([]rune(completion))
				le.redrawLine(line, cursor)
			} else if len(completions) > 1 {
				// Multiple completions - try common prefix completion first
				commonPrefix := findCommonPrefix(completions)

				// Find current word being completed
				wordStart := cursor
				for wordStart > 0 && line[wordStart-1] != ' ' {
					wordStart--
				}
				currentWord := string(line[wordStart:cursor])

				// If common prefix is longer than current word, complete to common prefix
				if len(commonPrefix) > len(currentWord) {
					newLine := append(line[:wordStart], []rune(commonPrefix)...)
					newLine = append(newLine, line[cursor:]...)
					line = newLine
					cursor = wordStart + len([]rune(commonPrefix))
					le.redrawLine(line, cursor)
				} else {
					// Show all completions
					fmt.Print("\n")
					le.showCompletions(completions)
					le.redrawLine(line, cursor)
				}
			}

		case '\x1b': // Escape sequence (arrow keys)
			seq, err := le.readEscapeSequence()
			if err != nil {
				// If we can't read the escape sequence properly, ignore it
				// This prevents malformed sequences from being added to the line
				continue
			}

			switch seq {
			case "A": // Up arrow - previous history
				if historyPos > 0 {
					if historyPos == le.history.Size() {
						originalLine = string(line)
					}
					historyPos--
					if historyPos < le.history.Size() {
						histCmd := le.history.GetAll()[historyPos]
						line = []rune(histCmd)
						cursor = len(line)
						le.redrawLine(line, cursor)
					}
				}

			case "B": // Down arrow - next history
				if historyPos < le.history.Size() {
					historyPos++
					if historyPos == le.history.Size() {
						line = []rune(originalLine)
					} else {
						histCmd := le.history.GetAll()[historyPos]
						line = []rune(histCmd)
					}
					cursor = len(line)
					le.redrawLine(line, cursor)
				}

			case "C": // Right arrow
				if cursor < len(line) {
					cursor++
					le.redrawLine(line, cursor)
				}

			case "D": // Left arrow
				if cursor > 0 {
					cursor--
					le.redrawLine(line, cursor)
				}

			case "H": // Home key
				cursor = 0
				le.redrawLine(line, cursor)

			case "F": // End key
				cursor = len(line)
				le.redrawLine(line, cursor)
			}

		default:
			// Regular character input
			if ch >= 32 && ch < 127 { // Printable ASCII
				line = append(line[:cursor], append([]rune{rune(ch)}, line[cursor:]...)...)
				cursor++
				le.redrawLine(line, cursor)
			}
		}
	}
}

// readEscapeSequence reads an escape sequence for arrow keys
func (le *LineEditor) readEscapeSequence() (string, error) {
	// Read the '[' character
	var buf [1]byte
	n, err := os.Stdin.Read(buf[:])
	if err != nil || n == 0 {
		return "", fmt.Errorf("incomplete escape sequence")
	}

	if buf[0] != '[' {
		return "", fmt.Errorf("unknown escape sequence")
	}

	// Read the actual key code
	n, err = os.Stdin.Read(buf[:])
	if err != nil || n == 0 {
		return "", fmt.Errorf("incomplete escape sequence")
	}

	return string(buf[0]), nil
}

// redrawLine redraws the current line and positions the cursor
func (le *LineEditor) redrawLine(line []rune, cursor int) {
	// Clear the line and move to beginning
	os.Stdout.WriteString("\033[2K\r")
	// Print prompt and line
	os.Stdout.WriteString("gosh> " + string(line))
	// Position cursor
	if cursor < len(line) {
		os.Stdout.WriteString(fmt.Sprintf("\033[%dD", len(line)-cursor))
	}
	// Ensure output is flushed
	os.Stdout.Sync()
}

// showCompletions displays available completions in a formatted way
func (le *LineEditor) showCompletions(completions []string) {
	const maxCols = 80
	const minColWidth = 12

	if len(completions) == 0 {
		return
	}

	// Find the maximum length for column width
	maxLen := 0
	for _, comp := range completions {
		if len(comp) > maxLen {
			maxLen = len(comp)
		}
	}

	colWidth := maxLen + 2
	if colWidth < minColWidth {
		colWidth = minColWidth
	}

	cols := maxCols / colWidth
	if cols < 1 {
		cols = 1
	}

	// Print completions in columns
	for i, comp := range completions {
		fmt.Printf("%-*s", colWidth, comp)
		if (i+1)%cols == 0 || i == len(completions)-1 {
			fmt.Print("\n")
		}
	}
}

// readLineSimple is a fallback for when raw mode is not available
func (le *LineEditor) readLineSimple() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(line, "\n"), nil
}

// Global line editor instance
var globalLineEditor *LineEditor

// SetHistory sets up the line editor with history support
func SetHistory(hist *history.History) {
	globalLineEditor = NewLineEditor(hist)
}

// ReadLine reads a line of input from stdin with a prompt and arrow key support
func ReadLine() (string, error) {
	// Use advanced line editing by default, fallback to simple mode if it fails
	if globalLineEditor != nil {
		result, err := globalLineEditor.ReadLineWithArrows()
		if err != nil && err.Error() != "interrupted" {
			// If advanced mode fails, fall back to simple mode
			fmt.Print("gosh> ")
			reader := bufio.NewReader(os.Stdin)
			line, _, err := reader.ReadLine()
			if err != nil {
				return "", err
			}
			return string(line), nil
		}
		return result, err
	}

	// Fallback to simple mode if line editor is not available
	fmt.Print("gosh> ")
	reader := bufio.NewReader(os.Stdin)
	line, _, err := reader.ReadLine()
	if err != nil {
		return "", err
	}
	return string(line), nil
}

// ParseLine parses a command line into arguments
func ParseLine(line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return []string{}
	}

	// Simple whitespace splitting for now
	// TODO: Handle quotes, escaping, etc.
	args := strings.Fields(line)
	return args
}

// ParseCommand parses a command line into a Command struct with redirection
func ParseCommand(line string) *Command {
	line = strings.TrimSpace(line)
	if line == "" {
		return &Command{}
	}

	cmd := &Command{}
	tokens := strings.Fields(line)

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		switch token {
		case ">":
			if i+1 < len(tokens) {
				cmd.OutputFile = tokens[i+1]
				cmd.AppendOutput = false
				i++ // skip the filename
			}
		case ">>":
			if i+1 < len(tokens) {
				cmd.OutputFile = tokens[i+1]
				cmd.AppendOutput = true
				i++ // skip the filename
			}
		case "<":
			if i+1 < len(tokens) {
				cmd.InputFile = tokens[i+1]
				i++ // skip the filename
			}
		case "&":
			cmd.Background = true
		default:
			cmd.Args = append(cmd.Args, token)
		}
	}

	// Expand environment variables in arguments
	cmd.Args = expandArgsVariables(cmd.Args)

	return cmd
}

// ParsePipeline parses a command line into a Pipeline with potential pipes
func ParsePipeline(line string) *Pipeline {
	line = strings.TrimSpace(line)
	if line == "" {
		return &Pipeline{}
	}

	// Filter out any escape sequences that might have gotten through
	// This prevents malformed input from causing panics
	if strings.Contains(line, "\x1b") || strings.Contains(line, "^[") {
		return &Pipeline{}
	}

	pipeline := &Pipeline{}

	// Check for background execution at the end
	if strings.HasSuffix(line, "&") {
		pipeline.Background = true
		line = strings.TrimSuffix(line, "&")
		line = strings.TrimSpace(line)
	}

	// Split by pipes
	pipeSegments := strings.Split(line, "|")

	for _, segment := range pipeSegments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}

		cmd := &Command{}
		tokens := strings.Fields(segment)

		for i := 0; i < len(tokens); i++ {
			token := tokens[i]

			switch token {
			case ">":
				if i+1 < len(tokens) {
					cmd.OutputFile = tokens[i+1]
					cmd.AppendOutput = false
					i++ // skip the filename
				}
			case ">>":
				if i+1 < len(tokens) {
					cmd.OutputFile = tokens[i+1]
					cmd.AppendOutput = true
					i++ // skip the filename
				}
			case "<":
				if i+1 < len(tokens) {
					cmd.InputFile = tokens[i+1]
					i++ // skip the filename
				}
			default:
				cmd.Args = append(cmd.Args, token)
			}
		}

		// Expand environment variables in arguments
		cmd.Args = expandArgsVariables(cmd.Args)

		pipeline.Commands = append(pipeline.Commands, cmd)
	}

	return pipeline
}

// ExpandVariables expands environment variables in a string
// Supports both $VAR and ${VAR} syntax
func ExpandVariables(s string) string {
	// Handle ${VAR} syntax
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	s = re.ReplaceAllStringFunc(s, func(match string) string {
		varName := match[2 : len(match)-1] // Remove ${ and }
		return os.Getenv(varName)
	})

	// Handle $VAR syntax (word boundaries)
	re = regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
	s = re.ReplaceAllStringFunc(s, func(match string) string {
		varName := match[1:] // Remove $
		return os.Getenv(varName)
	})

	return s
}

// expandArgsVariables expands environment variables in all arguments
func expandArgsVariables(args []string) []string {
	expanded := make([]string, len(args))
	for i, arg := range args {
		expanded[i] = ExpandVariables(arg)
	}
	return expanded
}
