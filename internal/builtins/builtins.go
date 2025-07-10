package builtins

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

var builtinCommands = map[string]func([]string) bool{
	"exit": exitCommand,
	"cd":   cdCommand,
	"pwd":  pwdCommand,
	"help": helpCommand,
	"env":  envCommand,
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
	fmt.Println("  cd [dir]     - Change directory")
	fmt.Println("  pwd          - Print working directory")
	fmt.Println("  env [VAR=val] - Show or set environment variables")
	fmt.Println("  help         - Show this help")
	fmt.Println("  exit         - Exit the shell")
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
