package executor

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/apriljarosz/gosh/internal/builtins"
	"github.com/apriljarosz/gosh/internal/input"
)

// Execute runs a command with the given arguments
// Returns false if the shell should exit
func Execute(args []string) bool {
	if len(args) == 0 {
		return true
	}

	command := args[0]

	// Check if it's a builtin command
	if builtins.IsBuiltin(command) {
		return builtins.Execute(command, args[1:])
	}

	// Execute external command
	cmd := exec.Command(command, args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gosh: %s: %v\n", command, err)
	}

	return true
}

// ExecuteCommand runs a parsed command with redirection support
// Returns false if the shell should exit
func ExecuteCommand(cmd *input.Command) bool {
	if len(cmd.Args) == 0 {
		return true
	}

	command := cmd.Args[0]

	// Check if it's a builtin command
	if builtins.IsBuiltin(command) {
		return builtins.Execute(command, cmd.Args[1:])
	}

	// Execute external command with redirection
	execCmd := exec.Command(command, cmd.Args[1:]...)

	// Handle input redirection
	if cmd.InputFile != "" {
		inputFile, err := os.Open(cmd.InputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gosh: %v\n", err)
			return true
		}
		defer inputFile.Close()
		execCmd.Stdin = inputFile
	} else {
		execCmd.Stdin = os.Stdin
	}

	// Handle output redirection
	if cmd.OutputFile != "" {
		var outputFile *os.File
		var err error

		if cmd.AppendOutput {
			outputFile, err = os.OpenFile(cmd.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		} else {
			outputFile, err = os.Create(cmd.OutputFile)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "gosh: %v\n", err)
			return true
		}
		defer outputFile.Close()
		execCmd.Stdout = outputFile
	} else {
		execCmd.Stdout = os.Stdout
	}

	execCmd.Stderr = os.Stderr

	// Handle background execution
	var err error
	if cmd.Background {
		err = execCmd.Start()
		if err == nil {
			fmt.Printf("[%d] %d\n", 1, execCmd.Process.Pid) // Simple job numbering
		}
	} else {
		err = execCmd.Run()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "gosh: %s: %v\n", command, err)
	}

	return true
}

// ExecutePipeline runs a pipeline of commands connected by pipes
// Returns false if the shell should exit
func ExecutePipeline(pipeline *input.Pipeline) bool {
	if len(pipeline.Commands) == 0 {
		return true
	}

	// Single command - use ExecuteCommand
	if len(pipeline.Commands) == 1 {
		cmd := pipeline.Commands[0]
		cmd.Background = pipeline.Background
		return ExecuteCommand(cmd)
	}

	// Multiple commands - set up pipes
	var cmds []*exec.Cmd

	for i, cmd := range pipeline.Commands {
		if len(cmd.Args) == 0 {
			continue
		}

		command := cmd.Args[0]

		// Check if it's a builtin command - builtins can't be piped easily
		if builtins.IsBuiltin(command) {
			fmt.Fprintf(os.Stderr, "gosh: cannot pipe builtin command: %s\n", command)
			return true
		}

		execCmd := exec.Command(command, cmd.Args[1:]...)

		// Handle input for first command
		if i == 0 {
			if cmd.InputFile != "" {
				inputFile, err := os.Open(cmd.InputFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "gosh: %v\n", err)
					return true
				}
				defer inputFile.Close()
				execCmd.Stdin = inputFile
			} else {
				execCmd.Stdin = os.Stdin
			}
		}

		// Handle output for last command
		if i == len(pipeline.Commands)-1 {
			if cmd.OutputFile != "" {
				var outputFile *os.File
				var err error

				if cmd.AppendOutput {
					outputFile, err = os.OpenFile(cmd.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
				} else {
					outputFile, err = os.Create(cmd.OutputFile)
				}

				if err != nil {
					fmt.Fprintf(os.Stderr, "gosh: %v\n", err)
					return true
				}
				defer outputFile.Close()
				execCmd.Stdout = outputFile
			} else {
				execCmd.Stdout = os.Stdout
			}
		}

		execCmd.Stderr = os.Stderr
		cmds = append(cmds, execCmd)
	}

	// Connect pipes between commands
	for i := 0; i < len(cmds)-1; i++ {
		pipe, err := cmds[i].StdoutPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "gosh: %v\n", err)
			return true
		}
		cmds[i+1].Stdin = pipe
	}

	// Start all commands
	for _, cmd := range cmds {
		err := cmd.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "gosh: %v\n", err)
			return true
		}
	}

	// Handle background execution
	if pipeline.Background {
		fmt.Printf("[%d] %d\n", 1, cmds[len(cmds)-1].Process.Pid)
		return true
	}

	// Wait for all commands to complete
	for _, cmd := range cmds {
		err := cmd.Wait()
		if err != nil {
			fmt.Fprintf(os.Stderr, "gosh: %v\n", err)
		}
	}

	return true
}
