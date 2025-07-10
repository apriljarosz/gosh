package input

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Command represents a parsed command with potential redirection
type Command struct {
	Args         []string
	InputFile    string
	OutputFile   string
	AppendOutput bool
	Background   bool
}

// Pipeline represents a series of commands connected by pipes
type Pipeline struct {
	Commands   []*Command
	Background bool
}

// ReadLine reads a line of input from stdin with a prompt
func ReadLine() (string, error) {
	fmt.Print("gosh> ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// Remove trailing newline
	line = strings.TrimSuffix(line, "\n")
	return line, nil
}

// ParseLine parses a command line into arguments
func ParseLine(line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return []string{}
	}

	// Simple whitespace splitting for now
	// TODO: Handle quotes, escaping, etc.
	args := strings.Fields(line)
	return args
}

// ParseCommand parses a command line into a Command struct with redirection
func ParseCommand(line string) *Command {
	line = strings.TrimSpace(line)
	if line == "" {
		return &Command{}
	}

	cmd := &Command{}
	tokens := strings.Fields(line)

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		switch token {
		case ">":
			if i+1 < len(tokens) {
				cmd.OutputFile = tokens[i+1]
				cmd.AppendOutput = false
				i++ // skip the filename
			}
		case ">>":
			if i+1 < len(tokens) {
				cmd.OutputFile = tokens[i+1]
				cmd.AppendOutput = true
				i++ // skip the filename
			}
		case "<":
			if i+1 < len(tokens) {
				cmd.InputFile = tokens[i+1]
				i++ // skip the filename
			}
		case "&":
			cmd.Background = true
		default:
			cmd.Args = append(cmd.Args, token)
		}
	}

	// Expand environment variables in arguments
	cmd.Args = expandArgsVariables(cmd.Args)

	return cmd
}

// ParsePipeline parses a command line into a Pipeline with potential pipes
func ParsePipeline(line string) *Pipeline {
	line = strings.TrimSpace(line)
	if line == "" {
		return &Pipeline{}
	}

	pipeline := &Pipeline{}

	// Check for background execution at the end
	if strings.HasSuffix(line, "&") {
		pipeline.Background = true
		line = strings.TrimSuffix(line, "&")
		line = strings.TrimSpace(line)
	}

	// Split by pipes
	pipeSegments := strings.Split(line, "|")

	for _, segment := range pipeSegments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}

		cmd := &Command{}
		tokens := strings.Fields(segment)

		for i := 0; i < len(tokens); i++ {
			token := tokens[i]

			switch token {
			case ">":
				if i+1 < len(tokens) {
					cmd.OutputFile = tokens[i+1]
					cmd.AppendOutput = false
					i++ // skip the filename
				}
			case ">>":
				if i+1 < len(tokens) {
					cmd.OutputFile = tokens[i+1]
					cmd.AppendOutput = true
					i++ // skip the filename
				}
			case "<":
				if i+1 < len(tokens) {
					cmd.InputFile = tokens[i+1]
					i++ // skip the filename
				}
			default:
				cmd.Args = append(cmd.Args, token)
			}
		}

		// Expand environment variables in arguments
		cmd.Args = expandArgsVariables(cmd.Args)

		pipeline.Commands = append(pipeline.Commands, cmd)
	}

	return pipeline
}

// ExpandVariables expands environment variables in a string
// Supports both $VAR and ${VAR} syntax
func ExpandVariables(s string) string {
	// Handle ${VAR} syntax
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	s = re.ReplaceAllStringFunc(s, func(match string) string {
		varName := match[2 : len(match)-1] // Remove ${ and }
		return os.Getenv(varName)
	})

	// Handle $VAR syntax (word boundaries)
	re = regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
	s = re.ReplaceAllStringFunc(s, func(match string) string {
		varName := match[1:] // Remove $
		return os.Getenv(varName)
	})

	return s
}

// expandArgsVariables expands environment variables in all arguments
func expandArgsVariables(args []string) []string {
	expanded := make([]string, len(args))
	for i, arg := range args {
		expanded[i] = ExpandVariables(arg)
	}
	return expanded
}
