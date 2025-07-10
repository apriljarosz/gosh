package input

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
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
	history     *history.History
	originalTty termios
	rawMode     bool
}

// NewLineEditor creates a new line editor with history support
func NewLineEditor(hist *history.History) *LineEditor {
	return &LineEditor{
		history: hist,
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

	// Create raw mode settings
	raw := le.originalTty
	raw.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	raw.Cflag &^= syscall.CSIZE | syscall.PARENB
	raw.Cflag |= syscall.CS8
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
		// Fall back to simple line reading if raw mode fails
		return le.readLineSimple()
	}
	defer le.disableRawMode()

	var line []rune
	var cursor int
	historyIndex := -1
	le.history.Reset()

	for {
		var buf [1]byte
		n, err := os.Stdin.Read(buf[:])
		if err != nil {
			return "", err
		}
		if n == 0 {
			continue
		}

		ch := buf[0]

		switch ch {
		case 13: // Enter
			fmt.Print("\r\n")
			return string(line), nil

		case 3: // Ctrl+C
			fmt.Print("^C\r\n")
			return "", fmt.Errorf("interrupted")

		case 4: // Ctrl+D (EOF)
			if len(line) == 0 {
				fmt.Print("\r\n")
				return "", fmt.Errorf("EOF")
			}

		case 127, 8: // Backspace
			if cursor > 0 {
				line = append(line[:cursor-1], line[cursor:]...)
				cursor--
				le.redrawLine(line, cursor)
			}

		case 27: // Escape sequence (arrow keys)
			seq, err := le.readEscapeSequence()
			if err != nil {
				continue
			}

			switch seq {
			case "A": // Up arrow
				if historyIndex == -1 {
					historyIndex = le.history.Size()
				}
				if historyIndex > 0 {
					historyIndex--
					if historyIndex < le.history.Size() {
						histCmd := le.history.GetAll()[historyIndex]
						line = []rune(histCmd)
						cursor = len(line)
						le.redrawLine(line, cursor)
					}
				}

			case "B": // Down arrow
				if historyIndex >= 0 {
					historyIndex++
					if historyIndex < le.history.Size() {
						histCmd := le.history.GetAll()[historyIndex]
						line = []rune(histCmd)
						cursor = len(line)
						le.redrawLine(line, cursor)
					} else {
						historyIndex = -1
						line = []rune{}
						cursor = 0
						le.redrawLine(line, cursor)
					}
				}

			case "C": // Right arrow
				if cursor < len(line) {
					cursor++
					fmt.Print("\033[C")
				}

			case "D": // Left arrow
				if cursor > 0 {
					cursor--
					fmt.Print("\033[D")
				}
			}

		default:
			if ch >= 32 && ch < 127 { // Printable characters
				line = append(line[:cursor], append([]rune{rune(ch)}, line[cursor:]...)...)
				cursor++
				le.redrawLine(line, cursor)
			}
		}
	}
}

// readEscapeSequence reads an escape sequence for arrow keys
func (le *LineEditor) readEscapeSequence() (string, error) {
	var buf [2]byte
	n, err := os.Stdin.Read(buf[:])
	if err != nil || n < 2 {
		return "", fmt.Errorf("incomplete escape sequence")
	}

	if buf[0] == '[' {
		return string(buf[1]), nil
	}

	return "", fmt.Errorf("unknown escape sequence")
}

// redrawLine redraws the current line and positions the cursor
func (le *LineEditor) redrawLine(line []rune, cursor int) {
	// Clear the line
	fmt.Print("\r\033[K")
	// Print prompt and line
	fmt.Print("gosh> " + string(line))
	// Position cursor
	if cursor < len(line) {
		fmt.Printf("\r\033[%dC", cursor+6) // 6 = len("gosh> ")
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
	if globalLineEditor != nil {
		return globalLineEditor.ReadLineWithArrows()
	}

	// Fallback to simple reading if no line editor is set up
	fmt.Print("gosh> ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(line, "\n"), nil
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
