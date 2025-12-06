package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/holydocs/holydocs/internal/adapters"
	"github.com/holydocs/holydocs/internal/adapters/primary/cli"
	"github.com/holydocs/holydocs/internal/config"
	"github.com/holydocs/holydocs/internal/core"
	do "github.com/samber/do/v2"
	"github.com/spf13/cobra"
)

const (
	appName        = "holydocs"
	appDescription = "generate system-architecture documentation"
	appLongDesc    = `HolyDOCs is a tool for generating docs from AsyncAPI, ServiceFile, etc.`
)

var (
	ErrCommandExecution = errors.New("command execution failed")
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	injector := do.New(
		core.Package,
		adapters.PrimaryPackage,
		adapters.SecondaryPackage,
		config.Package,
	)

	rootCmd := buildRootCommand(injector)

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		configFile, err := cmd.Flags().GetString("config")
		if err != nil {
			return fmt.Errorf("getting config file flag: %w", err)
		}

		do.ProvideValue(injector, config.ConfigFilePath(configFile))

		return nil
	}

	if err := rootCmd.Execute(); err != nil {
		return fmt.Errorf("%w: %w", ErrCommandExecution, err)
	}

	return nil
}

func buildRootCommand(injector do.Injector) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   appName,
		Short: appDescription,
		Long:  appLongDesc,
	}

	rootCmd.PersistentFlags().StringP("config", "c", "holydocs.yaml", "Path to YAML configuration file")

	cliCommand := do.MustInvoke[*cli.Command](injector)
	rootCmd.AddCommand(cliCommand.GetCommand())

	return rootCmd
}
