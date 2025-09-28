package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/holydocs/holydocs/cmd/holydocs/commands/docs"
	"github.com/spf13/cobra"
)

// Application constants.
const (
	appName        = "holydocs"
	appDescription = "generate system-architecture documentation"
	appLongDesc    = `HolyDOCs is a tool for generating docs from AsyncAPI, ServiceFile, etc.`
)

// Errors.
var (
	ErrCommandExecution = errors.New("command execution failed")
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run executes the main application logic.
func run() error {
	rootCmd := buildRootCommand()

	if err := rootCmd.Execute(); err != nil {
		return fmt.Errorf("%w: %w", ErrCommandExecution, err)
	}

	return nil
}

// buildRootCommand creates and configures the root cobra command.
func buildRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   appName,
		Short: appDescription,
		Long:  appLongDesc,
	}

	rootCmd.AddCommand(docs.NewCommand().GetCommand())

	return rootCmd
}
