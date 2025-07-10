package main

import (
	"fmt"
	"os"

	"github.com/apriljarosz/gosh/internal/builtins"
	"github.com/apriljarosz/gosh/internal/executor"
	"github.com/apriljarosz/gosh/internal/history"
	"github.com/apriljarosz/gosh/internal/input"
)

func main() {
	fmt.Println("Welcome to gosh - Go Shell")

	// Initialize history
	hist := history.New()
	builtins.SetHistory(hist)
	input.SetHistory(hist)

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
