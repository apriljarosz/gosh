package input

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Command
	}{
		{
			name:  "only redirection operators",
			input: "> < >>",
			expected: &Command{
				Args:       []string{},
				OutputFile: "<", // Current parsing behavior - last token becomes output
			},
		},
		{
			name:  "append redirection",
			input: "echo world >> file.txt",
			expected: &Command{
				Args:         []string{"echo", "world"},
				OutputFile:   "file.txt",
				AppendOutput: true,
			},
		},
		{
			name:  "input redirection",
			input: "wc -l < file.txt",
			expected: &Command{
				Args:      []string{"wc", "-l"},
				InputFile: "file.txt",
			},
		},
		{
			name:  "background execution",
			input: "sleep 5 &",
			expected: &Command{
				Args:       []string{"sleep", "5"},
				Background: true,
			},
		},
		{
			name:  "complex redirection",
			input: "sort < input.txt > output.txt",
			expected: &Command{
				Args:       []string{"sort"},
				InputFile:  "input.txt",
				OutputFile: "output.txt",
			},
		},
		{
			name:     "empty input",
			input:    "",
			expected: &Command{},
		},
		{
			name:     "whitespace only",
			input:    "   \t  ",
			expected: &Command{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCommand(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParsePipeline(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Pipeline
	}{
		{
			name:  "single command",
			input: "ls -la",
			expected: &Pipeline{
				Commands: []*Command{
					{Args: []string{"ls", "-la"}},
				},
			},
		},
		{
			name:  "simple pipe",
			input: "ls | wc -l",
			expected: &Pipeline{
				Commands: []*Command{
					{Args: []string{"ls"}},
					{Args: []string{"wc", "-l"}},
				},
			},
		},
		{
			name:  "multiple pipes",
			input: "ps aux | grep go | wc -l",
			expected: &Pipeline{
				Commands: []*Command{
					{Args: []string{"ps", "aux"}},
					{Args: []string{"grep", "go"}},
					{Args: []string{"wc", "-l"}},
				},
			},
		},
		{
			name:  "pipe with redirection",
			input: "ls | sort > output.txt",
			expected: &Pipeline{
				Commands: []*Command{
					{Args: []string{"ls"}},
					{Args: []string{"sort"}, OutputFile: "output.txt"},
				},
			},
		},
		{
			name:  "background pipeline",
			input: "find . -name '*.go' | wc -l &",
			expected: &Pipeline{
				Commands: []*Command{
					{Args: []string{"find", ".", "-name", "'*.go'"}},
					{Args: []string{"wc", "-l"}},
				},
				Background: true,
			},
		},
		{
			name:     "empty pipeline",
			input:    "",
			expected: &Pipeline{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParsePipeline(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpandVariables(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_VAR", "hello")
	os.Setenv("USER_NAME", "testuser")
	os.Setenv("PATH_VAR", "/usr/bin:/bin")
	defer func() {
		os.Unsetenv("TEST_VAR")
		os.Unsetenv("USER_NAME")
		os.Unsetenv("PATH_VAR")
	}()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple variable",
			input:    "echo $TEST_VAR",
			expected: "echo hello",
		},
		{
			name:     "braced variable",
			input:    "echo ${TEST_VAR}_world",
			expected: "echo hello_world",
		},
		{
			name:     "multiple variables",
			input:    "$USER_NAME uses $TEST_VAR",
			expected: "testuser uses hello",
		},
		{
			name:     "mixed syntax",
			input:    "${USER_NAME} says $TEST_VAR",
			expected: "testuser says hello",
		},
		{
			name:     "undefined variable",
			input:    "echo $UNDEFINED_VAR",
			expected: "echo ",
		},
		{
			name:     "no variables",
			input:    "echo hello world",
			expected: "echo hello world",
		},
		{
			name:     "variable with path",
			input:    "export PATH=$PATH_VAR",
			expected: "export PATH=/usr/bin:/bin",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "dollar without variable",
			input:    "echo $ and $$ and $123",
			expected: "echo $ and $$ and $123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandVariables(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseLineBasic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple command",
			input:    "ls -la",
			expected: []string{"ls", "-la"},
		},
		{
			name:     "multiple arguments",
			input:    "find . -name '*.go' -type f",
			expected: []string{"find", ".", "-name", "'*.go'", "-type", "f"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "whitespace only",
			input:    "   \t  \n  ",
			expected: []string{},
		},
		{
			name:     "extra whitespace",
			input:    "  ls   -la   ",
			expected: []string{"ls", "-la"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseLine(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpandArgsVariables(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_VAR", "hello")
	os.Setenv("NUM_VAR", "42")
	defer func() {
		os.Unsetenv("TEST_VAR")
		os.Unsetenv("NUM_VAR")
	}()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "expand single variable",
			input:    []string{"echo", "$TEST_VAR"},
			expected: []string{"echo", "hello"},
		},
		{
			name:     "expand multiple variables",
			input:    []string{"echo", "$TEST_VAR", "$NUM_VAR"},
			expected: []string{"echo", "hello", "42"},
		},
		{
			name:     "no variables",
			input:    []string{"ls", "-la"},
			expected: []string{"ls", "-la"},
		},
		{
			name:     "mixed variables and literals",
			input:    []string{"echo", "$TEST_VAR", "world", "$NUM_VAR"},
			expected: []string{"echo", "hello", "world", "42"},
		},
		{
			name:     "empty args",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandArgsVariables(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Edge case tests
func TestParseCommandEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Command
	}{
		{
			name:  "multiple redirections (last wins)",
			input: "echo hello > file1.txt > file2.txt",
			expected: &Command{
				Args:       []string{"echo", "hello"},
				OutputFile: "file2.txt",
			},
		},
		{
			name:  "redirection without filename",
			input: "echo hello >",
			expected: &Command{
				Args: []string{"echo", "hello"},
			},
		},
		{
			name:  "background with other operators",
			input: "echo hello > file.txt &",
			expected: &Command{
				Args:       []string{"echo", "hello"},
				OutputFile: "file.txt",
				Background: true,
			},
		},
		{
			name:  "only redirection operators",
			input: "> < >>",
			expected: &Command{
				Args:       []string{},
				OutputFile: "<", // Current parsing behavior - last token becomes output
			},
		},
		{
			name:  "command with equals sign",
			input: "env VAR=value",
			expected: &Command{
				Args: []string{"env", "VAR=value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCommand(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Tab completion tests
func TestCompletionEngine_Complete(t *testing.T) {
	ce := NewCompletionEngine()

	tests := []struct {
		name     string
		line     string
		cursor   int
		contains []string // Commands that should be in the result
		excludes []string // Commands that should not be in the result
	}{
		{
			name:     "complete builtin command",
			line:     "h",
			cursor:   1,
			contains: []string{"help", "history"},
			excludes: []string{"cd", "exit"},
		},
		{
			name:     "complete exact builtin",
			line:     "help",
			cursor:   4,
			contains: []string{"help"},
			excludes: []string{"history", "cd"},
		},
		{
			name:     "complete partial command",
			line:     "ex",
			cursor:   2,
			contains: []string{"exit"},
			excludes: []string{"help", "cd"},
		},
		{
			name:     "no matches for nonsense",
			line:     "xyznonsense",
			cursor:   11,
			contains: []string{},
			excludes: []string{"help", "cd", "exit"},
		},
		{
			name:     "empty line includes builtins",
			line:     "",
			cursor:   0,
			contains: []string{"cd", "env", "exit", "help", "history", "pwd"},
			excludes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ce.Complete(tt.line, tt.cursor)

			// Check that all expected commands are present
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected, "Expected %s to be in completion results", expected)
			}

			// Check that excluded commands are not present
			for _, excluded := range tt.excludes {
				assert.NotContains(t, result, excluded, "Expected %s to NOT be in completion results", excluded)
			}
		})
	}
}
func TestCompletionEngine_CompleteCommand(t *testing.T) {
	ce := NewCompletionEngine()

	tests := []struct {
		name     string
		prefix   string
		contains []string // Builtin commands that should be in the result
		excludes []string // Builtin commands that should not be in the result
	}{
		{
			name:     "complete h commands",
			prefix:   "h",
			contains: []string{"help", "history"},
			excludes: []string{"cd", "env", "exit", "pwd"},
		},
		{
			name:     "complete e commands",
			prefix:   "e",
			contains: []string{"env", "exit"},
			excludes: []string{"cd", "help", "history", "pwd"},
		},
		{
			name:     "complete exact match",
			prefix:   "pwd",
			contains: []string{"pwd"},
			excludes: []string{"cd", "env", "exit", "help", "history"},
		},
		{
			name:     "no builtin matches",
			prefix:   "xyznonsense",
			contains: []string{},
			excludes: []string{"cd", "env", "exit", "help", "history", "pwd"},
		},
		{
			name:     "empty prefix includes all builtins",
			prefix:   "",
			contains: []string{"cd", "env", "exit", "help", "history", "pwd"},
			excludes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ce.completeCommand(tt.prefix)

			// Filter to only builtin commands for predictable testing
			var builtinResults []string
			for _, cmd := range result {
				for _, builtin := range ce.builtinCommands {
					if cmd == builtin {
						builtinResults = append(builtinResults, cmd)
						break
					}
				}
			}

			// Check that all expected builtins are present
			for _, expected := range tt.contains {
				assert.Contains(t, builtinResults, expected, "Expected builtin %s to be in completion results", expected)
			}

			// Check that excluded builtins are not present
			for _, excluded := range tt.excludes {
				assert.NotContains(t, builtinResults, excluded, "Expected builtin %s to NOT be in completion results", excluded)
			}
		})
	}
}
func TestNewCompletionEngine(t *testing.T) {
	ce := NewCompletionEngine()

	assert.NotNil(t, ce)
	assert.NotNil(t, ce.builtinCommands)
	assert.Contains(t, ce.builtinCommands, "cd")
	assert.Contains(t, ce.builtinCommands, "pwd")
	assert.Contains(t, ce.builtinCommands, "exit")
	assert.Contains(t, ce.builtinCommands, "help")
	assert.Contains(t, ce.builtinCommands, "env")
	assert.Contains(t, ce.builtinCommands, "history")
}
