package main

import (
	"fmt"
	"os"

	"github.com/apriljarosz/gosh/internal/executor"
	"github.com/apriljarosz/gosh/internal/input"
)

func main() {
	fmt.Println("Welcome to gosh - Go Shell")

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

		pipeline := input.ParsePipeline(line)
		if len(pipeline.Commands) == 0 {
			continue
		}

		if !executor.ExecutePipeline(pipeline) {
			break
		}
	}
}
