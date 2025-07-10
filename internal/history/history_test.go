package history

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHistoryBasic(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	h := &History{
		commands:    make([]string, 0),
		currentPos:  -1,
		maxSize:     10,
		historyPath: filepath.Join(tempDir, ".test_history"),
	}

	// Test adding commands
	h.Add("echo hello")
	h.Add("ls -la")
	h.Add("pwd")

	assert.Equal(t, 3, h.Size())
	assert.Equal(t, []string{"echo hello", "ls -la", "pwd"}, h.GetAll())
}

func TestHistoryNavigation(t *testing.T) {
	h := &History{
		commands:   []string{"cmd1", "cmd2", "cmd3"},
		currentPos: 3,
		maxSize:    10,
	}

	// Test previous navigation
	assert.Equal(t, "cmd3", h.Previous())
	assert.Equal(t, "cmd2", h.Previous())
	assert.Equal(t, "cmd1", h.Previous())
	assert.Equal(t, "cmd1", h.Previous()) // Should stay at first

	// Test next navigation
	assert.Equal(t, "cmd2", h.Next())
	assert.Equal(t, "cmd3", h.Next())
	assert.Equal(t, "", h.Next()) // Past end should return empty
}

func TestHistoryDuplicates(t *testing.T) {
	h := &History{
		commands:   make([]string, 0),
		currentPos: 0,
		maxSize:    10,
	}

	h.Add("echo hello")
	h.Add("echo hello") // Duplicate
	h.Add("ls")
	h.Add("ls") // Duplicate

	// Should only have unique consecutive commands
	assert.Equal(t, 2, h.Size())
	assert.Equal(t, []string{"echo hello", "ls"}, h.GetAll())
}

func TestHistoryMaxSize(t *testing.T) {
	h := &History{
		commands:   make([]string, 0),
		currentPos: 0,
		maxSize:    3,
	}

	h.Add("cmd1")
	h.Add("cmd2")
	h.Add("cmd3")
	h.Add("cmd4") // Should remove cmd1

	assert.Equal(t, 3, h.Size())
	assert.Equal(t, []string{"cmd2", "cmd3", "cmd4"}, h.GetAll())
}

func TestHistorySaveLoad(t *testing.T) {
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, ".test_history")

	// Create and populate history
	h1 := &History{
		commands:    make([]string, 0),
		currentPos:  0,
		maxSize:     10,
		historyPath: historyPath,
	}

	h1.Add("echo test1")
	h1.Add("echo test2")
	h1.Add("pwd")

	// Save history
	err := h1.Save()
	assert.NoError(t, err)

	// Create new history and load
	h2 := &History{
		commands:    make([]string, 0),
		currentPos:  0,
		maxSize:     10,
		historyPath: historyPath,
	}

	err = h2.Load()
	assert.NoError(t, err)

	assert.Equal(t, h1.GetAll(), h2.GetAll())
	assert.Equal(t, 3, h2.Size())
}

func TestHistoryEmpty(t *testing.T) {
	h := &History{
		commands:   make([]string, 0),
		currentPos: 0,
		maxSize:    10,
	}

	assert.Equal(t, "", h.Previous())
	assert.Equal(t, "", h.Next())
	assert.Equal(t, 0, h.Size())
	assert.Equal(t, []string{}, h.GetAll())
}

func TestHistoryReset(t *testing.T) {
	h := &History{
		commands:   []string{"cmd1", "cmd2", "cmd3"},
		currentPos: 1,
		maxSize:    10,
	}

	h.Reset()
	assert.Equal(t, 3, h.currentPos) // Should be at end
}

func TestHistoryAddEmpty(t *testing.T) {
	h := &History{
		commands:   make([]string, 0),
		currentPos: 0,
		maxSize:    10,
	}

	h.Add("")
	h.Add("   ")
	h.Add("\t\n")

	assert.Equal(t, 0, h.Size())
	assert.Equal(t, []string{}, h.GetAll())
}
