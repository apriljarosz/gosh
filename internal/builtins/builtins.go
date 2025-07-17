package builtins

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/apriljarosz/gosh/internal/history"
	"github.com/apriljarosz/gosh/internal/jobs"
)

var builtinCommands = map[string]func([]string) bool{
	"exit":    exitCommand,
	"cd":      cdCommand,
	"pwd":     pwdCommand,
	"help":    helpCommand,
	"env":     envCommand,
	"history": historyCommand,
	"jobs":    jobsCommand,
	"fg":      fgCommand,
	"bg":      bgCommand,
}

// Global history instance - will be set by main
var globalHistory *history.History

// Global job manager instance - will be set by main
var globalJobManager *jobs.JobManager

// SetHistory sets the global history instance
func SetHistory(h *history.History) {
	globalHistory = h
}

// SetJobManager sets the global job manager instance
func SetJobManager(jm *jobs.JobManager) {
	globalJobManager = jm
}

// IsBuiltin checks if a command is a builtin
func IsBuiltin(command string) bool {
	_, exists := builtinCommands[command]
	return exists
}

// Execute runs a builtin command
// Returns false if the shell should exit
func Execute(command string, args []string) bool {
	if fn, exists := builtinCommands[command]; exists {
		return fn(args)
	}
	return true
}

func exitCommand(args []string) bool {
	fmt.Println("Goodbye!")
	return false
}

func cdCommand(args []string) bool {
	var dir string
	if len(args) == 0 {
		// Change to home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cd: %v\n", err)
			return true
		}
		dir = home
	} else {
		dir = args[0]
	}

	err := os.Chdir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cd: %v\n", err)
	}
	return true
}

func historyCommand(args []string) bool {
	if globalHistory == nil {
		fmt.Fprintf(os.Stderr, "history: history not available\n")
		return true
	}

	commands := globalHistory.GetAll()
	if len(commands) == 0 {
		return true
	}

	// Default to showing last 20 commands
	numToShow := 20
	if len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil && n > 0 {
			numToShow = n
		}
	}

	// Show the last numToShow commands
	start := len(commands) - numToShow
	if start < 0 {
		start = 0
	}

	for i := start; i < len(commands); i++ {
		fmt.Printf("%4d  %s\n", i+1, commands[i])
	}

	return true
}

func pwdCommand(args []string) bool {
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pwd: %v\n", err)
		return true
	}
	fmt.Println(pwd)
	return true
}

func helpCommand(args []string) bool {
	fmt.Println("gosh - Go Shell")
	fmt.Println("Built-in commands:")
	fmt.Println("  cd [dir]      - Change directory")
	fmt.Println("  pwd           - Print working directory")
	fmt.Println("  env [VAR=val] - Show or set environment variables")
	fmt.Println("  history [n]   - Show command history")
	fmt.Println("  jobs          - Show active jobs")
	fmt.Println("  fg <job_id>   - Bring job to foreground")
	fmt.Println("  bg <job_id>   - Send job to background")
	fmt.Println("  help          - Show this help")
	fmt.Println("  exit          - Exit the shell")
	return true
}

func envCommand(args []string) bool {
	if len(args) == 0 {
		// Show all environment variables
		environ := os.Environ()
		sort.Strings(environ)
		for _, env := range environ {
			fmt.Println(env)
		}
		return true
	}

	// Set environment variables
	for _, arg := range args {
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				err := os.Setenv(parts[0], parts[1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "env: %v\n", err)
				}
			}
		} else {
			// Show specific variable
			value := os.Getenv(arg)
			if value != "" {
				fmt.Printf("%s=%s\n", arg, value)
			}
		}
	}

	return true
}

func jobsCommand(args []string) bool {
	if globalJobManager == nil {
		fmt.Fprintf(os.Stderr, "jobs: job manager not available\n")
		return true
	}

	globalJobManager.PrintJobs()
	return true
}

func fgCommand(args []string) bool {
	if globalJobManager == nil {
		fmt.Fprintf(os.Stderr, "fg: job manager not available\n")
		return true
	}

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "fg: usage: fg <job_id>\n")
		return true
	}

	jobID, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "fg: invalid job ID: %s\n", args[0])
		return true
	}

	err = globalJobManager.BringToForeground(jobID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fg: %v\n", err)
	}

	return true
}

func bgCommand(args []string) bool {
	if globalJobManager == nil {
		fmt.Fprintf(os.Stderr, "bg: job manager not available\n")
		return true
	}

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "bg: usage: bg <job_id>\n")
		return true
	}

	jobID, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "bg: invalid job ID: %s\n", args[0])
		return true
	}

	err = globalJobManager.SendToBackground(jobID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bg: %v\n", err)
	}

	return true
}
