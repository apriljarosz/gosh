package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/apriljarosz/gosh/internal/builtins"
	"github.com/apriljarosz/gosh/internal/executor"
	"github.com/apriljarosz/gosh/internal/history"
	"github.com/apriljarosz/gosh/internal/input"
	"github.com/apriljarosz/gosh/internal/jobs"
)

func main() {
	// Set up signal handling - ignore SIGINT for the shell itself
	// Child processes will handle their own signals
	signal.Ignore(syscall.SIGINT)

	// Set terminal to cooked mode to handle line endings properly
	fmt.Print("\033[?1049l") // Exit alternate screen if in it
	fmt.Print("\033[0m")     // Reset all attributes

	fmt.Println("Welcome to gosh - Go Shell")
	// Initialize history
	hist := history.New()
	builtins.SetHistory(hist)
	input.SetHistory(hist)

	// Initialize readline with history
	if err := input.InitReadline(hist); err != nil {
		fmt.Printf("Warning: Could not initialize readline: %v\n", err)
	}
	defer input.CloseReadline()

	// Initialize job manager
	jobManager := jobs.NewJobManager()
	builtins.SetJobManager(jobManager)

	// Save history on exit
	defer hist.Save()

	for {
		line, err := input.ReadLine()
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println()
				break
			}
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			continue
		}

		if line == "" {
			continue
		}

		// Add command to history
		hist.Add(line)

		pipeline := input.ParsePipeline(line)
		if len(pipeline.Commands) == 0 {
			continue
		}

		if !executor.ExecutePipeline(pipeline) {
			break
		}
	}
}
