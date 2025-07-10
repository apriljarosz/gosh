package builtins

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBuiltin(t *testing.T) {
	tests := []struct {
		command  string
		expected bool
	}{
		{"cd", true},
		{"pwd", true},
		{"exit", true},
		{"help", true},
		{"env", true},
		{"ls", false},
		{"echo", false},
		{"grep", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := IsBuiltin(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecute(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		args     []string
		expected bool
	}{
		{
			name:     "exit command",
			command:  "exit",
			args:     []string{},
			expected: false, // Should return false to exit shell
		},
		{
			name:     "pwd command",
			command:  "pwd",
			args:     []string{},
			expected: true,
		},
		{
			name:     "help command",
			command:  "help",
			args:     []string{},
			expected: true,
		},
		{
			name:     "unknown builtin",
			command:  "unknown",
			args:     []string{},
			expected: true, // Should return true for unknown commands
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Execute(tt.command, tt.args)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPwdCommand(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute pwd command
	result := pwdCommand([]string{})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := strings.TrimSpace(buf.String())

	// Verify result
	assert.True(t, result)
	assert.NotEmpty(t, output)

	// Verify output matches current directory
	expectedPwd, _ := os.Getwd()
	assert.Equal(t, expectedPwd, output)
}

func TestCdCommand(t *testing.T) {
	// Save current directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir) // Restore at end

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "change to home directory",
			args:        []string{},
			expectError: false,
		},
		{
			name:        "change to /tmp",
			args:        []string{"/tmp"},
			expectError: false,
		},
		{
			name:        "change to non-existent directory",
			args:        []string{"/non/existent/directory"},
			expectError: true,
		},
		{
			name:        "change to current directory",
			args:        []string{"."},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr for error cases
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			result := cdCommand(tt.args)

			w.Close()
			os.Stderr = oldStderr

			var buf bytes.Buffer
			io.Copy(&buf, r)
			errorOutput := buf.String()

			// Should always return true (don't exit shell on cd errors)
			assert.True(t, result)

			if tt.expectError {
				assert.Contains(t, errorOutput, "cd:")
			} else {
				assert.Empty(t, errorOutput)
			}
		})
	}
}

func TestHelpCommand(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := helpCommand([]string{})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.True(t, result)
	assert.Contains(t, output, "gosh - Go Shell")
	assert.Contains(t, output, "Built-in commands:")
	assert.Contains(t, output, "cd")
	assert.Contains(t, output, "pwd")
	assert.Contains(t, output, "env")
	assert.Contains(t, output, "help")
	assert.Contains(t, output, "exit")
}

func TestEnvCommand(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_ENV_VAR", "test_value")
	os.Setenv("ANOTHER_VAR", "another_value")
	defer func() {
		os.Unsetenv("TEST_ENV_VAR")
		os.Unsetenv("ANOTHER_VAR")
	}()

	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		expectError    bool
	}{
		{
			name:           "show specific variable",
			args:           []string{"TEST_ENV_VAR"},
			expectedOutput: "TEST_ENV_VAR=test_value",
			expectError:    false,
		},
		{
			name:           "set new variable",
			args:           []string{"NEW_VAR=new_value"},
			expectedOutput: "",
			expectError:    false,
		},
		{
			name:           "show non-existent variable",
			args:           []string{"NON_EXISTENT"},
			expectedOutput: "",
			expectError:    false,
		},
		{
			name:           "multiple operations",
			args:           []string{"SET_VAR=value", "TEST_ENV_VAR"},
			expectedOutput: "TEST_ENV_VAR=test_value",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			rOut, wOut, _ := os.Pipe()
			rErr, wErr, _ := os.Pipe()
			os.Stdout = wOut
			os.Stderr = wErr

			result := envCommand(tt.args)

			wOut.Close()
			wErr.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			var bufOut, bufErr bytes.Buffer
			io.Copy(&bufOut, rOut)
			io.Copy(&bufErr, rErr)

			output := strings.TrimSpace(bufOut.String())
			errorOutput := bufErr.String()

			assert.True(t, result)

			if tt.expectError {
				assert.NotEmpty(t, errorOutput)
			} else {
				assert.Empty(t, errorOutput)
			}

			if tt.expectedOutput != "" {
				assert.Contains(t, output, tt.expectedOutput)
			}

			// Verify variable was actually set for set operations
			for _, arg := range tt.args {
				if strings.Contains(arg, "=") {
					parts := strings.SplitN(arg, "=", 2)
					if len(parts) == 2 {
						assert.Equal(t, parts[1], os.Getenv(parts[0]))
						os.Unsetenv(parts[0]) // Clean up
					}
				}
			}
		})
	}
}

func TestEnvCommandShowAll(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := envCommand([]string{})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.True(t, result)
	assert.NotEmpty(t, output)

	// Should contain some common environment variables
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Greater(t, len(lines), 0)

	// Each line should be in KEY=VALUE format
	for _, line := range lines {
		if line != "" {
			assert.Contains(t, line, "=")
		}
	}
}

func TestExitCommand(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := exitCommand([]string{})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := strings.TrimSpace(buf.String())

	assert.False(t, result) // Should return false to exit shell
	assert.Equal(t, "Goodbye!", output)
}
