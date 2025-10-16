package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/holydocs/holydocs/cmd/holydocs/commands/docs"
	"github.com/holydocs/holydocs/internal/config"
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

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		return loadConfig(cmd)
	}

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

	rootCmd.PersistentFlags().StringP("config", "c", "holydocs.yaml", "Path to YAML configuration file")

	rootCmd.AddCommand(docs.NewCommand().GetCommand())

	return rootCmd
}

// loadConfig loads configuration and stores it in the command context.
func loadConfig(cmd *cobra.Command) error {
	configFile, err := cmd.Flags().GetString("config")
	if err != nil {
		return fmt.Errorf("getting config file flag: %w", err)
	}

	cfg, err := config.Load(context.Background(), configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	ctx := config.WithContext(cmd.Context(), cfg)
	cmd.SetContext(ctx)

	return nil
}
