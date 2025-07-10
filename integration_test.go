package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShellIntegration(t *testing.T) {
	// Build the shell first
	buildCmd := exec.Command("go", "build", "-o", "gosh_test")
	err := buildCmd.Run()
	assert.NoError(t, err, "Failed to build shell")
	defer os.Remove("gosh_test") // Clean up

	tests := []struct {
		name           string
		input          string
		expectedOutput string
		expectFiles    []string
	}{
		{
			name:           "basic command",
			input:          "echo hello\nexit\n",
			expectedOutput: "hello",
		},
		{
			name:           "builtin pwd",
			input:          "pwd\nexit\n",
			expectedOutput: "", // Will contain current directory
		},
		{
			name:           "output redirection",
			input:          "echo 'test content' > test_output.txt\nexit\n",
			expectedOutput: "",
			expectFiles:    []string{"test_output.txt"},
		},
		{
			name:           "environment variables",
			input:          "echo $USER\nexit\n",
			expectedOutput: "april", // Use existing USER env var
		},
		{
			name:           "pipe commands",
			input:          "echo 'line1' | wc -l\nexit\n",
			expectedOutput: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for this test
			tempDir, err := ioutil.TempDir("", "gosh_test_")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Change to temp directory
			originalDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(originalDir)

			// Run the shell with input
			cmd := exec.Command(originalDir + "/gosh_test")
			cmd.Stdin = strings.NewReader(tt.input)

			output, err := cmd.CombinedOutput()
			assert.NoError(t, err, "Shell execution failed: %s", string(output))

			outputStr := string(output)

			// Check expected output (if specified)
			if tt.expectedOutput != "" {
				assert.Contains(t, outputStr, tt.expectedOutput)
			}

			// Check expected files
			for _, filename := range tt.expectFiles {
				_, err := os.Stat(filename)
				assert.NoError(t, err, "Expected file %s was not created", filename)

				// Clean up the file
				os.Remove(filename)
			}
		})
	}
}

func TestShellBackgroundJobs(t *testing.T) {
	// Build the shell first
	buildCmd := exec.Command("go", "build", "-o", "gosh_test")
	err := buildCmd.Run()
	assert.NoError(t, err, "Failed to build shell")
	defer os.Remove("gosh_test")

	// Test background job
	cmd := exec.Command("./gosh_test")
	cmd.Stdin = strings.NewReader("sleep 1 &\nexit\n")

	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Contains(t, string(output), "[1]") // Job number

	// Should complete quickly since sleep runs in background
	assert.Less(t, duration, 2*time.Second)
}

func TestShellErrorHandling(t *testing.T) {
	// Build the shell first
	buildCmd := exec.Command("go", "build", "-o", "gosh_test")
	err := buildCmd.Run()
	assert.NoError(t, err, "Failed to build shell")
	defer os.Remove("gosh_test")

	tests := []struct {
		name          string
		input         string
		expectError   bool
		errorContains string
	}{
		{
			name:          "invalid command",
			input:         "nonexistentcommand123\nexit\n",
			expectError:   true,
			errorContains: "gosh:",
		},
		{
			name:          "cd to non-existent directory",
			input:         "cd /non/existent/path\nexit\n",
			expectError:   true,
			errorContains: "cd:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./gosh_test")
			cmd.Stdin = strings.NewReader(tt.input)

			output, _ := cmd.CombinedOutput()
			outputStr := string(output)

			if tt.expectError {
				assert.Contains(t, outputStr, tt.errorContains)
			}
		})
	}
}
