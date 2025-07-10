package history

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxHistorySize = 1000
	historyFile    = ".gosh_history"
)

// History manages command history storage and retrieval
type History struct {
	commands    []string
	currentPos  int
	maxSize     int
	historyPath string
}

// New creates a new History instance
func New() *History {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	h := &History{
		commands:    make([]string, 0),
		currentPos:  -1,
		maxSize:     maxHistorySize,
		historyPath: filepath.Join(homeDir, historyFile),
	}

	h.Load()
	return h
}

// Add adds a command to history
func (h *History) Add(command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}

	// Don't add duplicate consecutive commands
	if len(h.commands) > 0 && h.commands[len(h.commands)-1] == command {
		h.currentPos = len(h.commands)
		return
	}

	h.commands = append(h.commands, command)

	// Trim history if it exceeds max size
	if len(h.commands) > h.maxSize {
		h.commands = h.commands[len(h.commands)-h.maxSize:]
	}

	h.currentPos = len(h.commands)
}

// Previous returns the previous command in history
func (h *History) Previous() string {
	if len(h.commands) == 0 {
		return ""
	}

	if h.currentPos > 0 {
		h.currentPos--
	}

	if h.currentPos >= 0 && h.currentPos < len(h.commands) {
		return h.commands[h.currentPos]
	}

	return ""
}

// Next returns the next command in history
func (h *History) Next() string {
	if len(h.commands) == 0 {
		return ""
	}

	if h.currentPos < len(h.commands) {
		h.currentPos++
	}

	if h.currentPos >= 0 && h.currentPos < len(h.commands) {
		return h.commands[h.currentPos]
	}

	// If we're past the end, return empty string
	return ""
}

// Reset resets the current position to the end of history
func (h *History) Reset() {
	h.currentPos = len(h.commands)
}

// GetAll returns all commands in history
func (h *History) GetAll() []string {
	return append([]string{}, h.commands...)
}

// Size returns the number of commands in history
func (h *History) Size() int {
	return len(h.commands)
}

// Load loads history from file
func (h *History) Load() error {
	file, err := os.Open(h.historyPath)
	if err != nil {
		// File doesn't exist, that's okay
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			h.commands = append(h.commands, line)
		}
	}

	// Trim to max size if needed
	if len(h.commands) > h.maxSize {
		h.commands = h.commands[len(h.commands)-h.maxSize:]
	}

	h.currentPos = len(h.commands)
	return scanner.Err()
}

// Save saves history to file
func (h *History) Save() error {
	file, err := os.Create(h.historyPath)
	if err != nil {
		return fmt.Errorf("failed to create history file: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, command := range h.commands {
		if _, err := writer.WriteString(command + "\n"); err != nil {
			return fmt.Errorf("failed to write to history file: %v", err)
		}
	}

	return nil
}
