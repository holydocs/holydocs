// Package config provides configuration management for HolyDOCs using aconfig.
// It supports loading configuration from flags, environment variables, and YAML files
// with a priority order: flags > environment variables > YAML file.
//
//nolint:lll
package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigyaml"
)

// Default values for configuration.
const (
	defaultTitle      = "HolyDOCs"
	defaultInputDir   = "."
	defaultOutputDir  = "docs"
	defaultGlobalName = "Internal Services"
)

// Config represents the complete configuration for HolyDOCs.
type Config struct {
	Input  Input  `yaml:"input"`
	Output Output `yaml:"output"`
}

// Input represents input configuration for HolyDOCs.
type Input struct {
	Directory     string   `yaml:"dir" default:"." usage:"Directory to scan for AsyncAPI and ServiceFile files"`
	AsyncAPIFiles []string `yaml:"asyncapi_files" usage:"Comma-separated list of AsyncAPI specification files"`
	ServiceFiles  []string `yaml:"service_files" usage:"Comma-separated list of ServiceFile specification files"`
}

// Output represents output configuration for HolyDOCs.
type Output struct {
	Directory  string `yaml:"directory" default:"docs" usage:"Directory where documentation will be generated"`
	Title      string `yaml:"title" default:"HolyDOCs" usage:"Title for the generated documentation"`
	GlobalName string `yaml:"global_name" default:"Internal Services" usage:"Name used for grouping internal services in diagrams"`
}

// Load loads configuration from multiple sources in priority order:
// 1. Environment variables
// 2. YAML configuration file.
func Load(_ context.Context, configFile string) (*Config, error) {
	cfg := &Config{
		Input: Input{
			Directory: defaultInputDir,
		},
		Output: Output{
			Directory:  defaultOutputDir,
			Title:      defaultTitle,
			GlobalName: defaultGlobalName,
		},
	}

	loaderConfig := aconfig.Config{
		EnvPrefix: "HOLYDOCS",
		SkipFlags: true,
		FileDecoders: map[string]aconfig.FileDecoder{
			".yaml": aconfigyaml.New(),
			".yml":  aconfigyaml.New(),
		},
		AllowUnknownFields: true,
	}

	if configFile != "" {
		loaderConfig.Files = []string{configFile}
	}

	loader := aconfig.LoaderFor(cfg, loaderConfig)

	if err := loader.Load(); err != nil {
		return nil, fmt.Errorf("loading configuration: %w", err)
	}

	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func validateConfig(cfg *Config) error {
	if cfg.Output.Title == "" {
		return errors.New("documentation title cannot be empty")
	}

	if cfg.Output.Directory == "" {
		return errors.New("output directory cannot be empty")
	}

	// Validate that at least one input source is provided
	if cfg.Input.Directory == "" &&
		len(cfg.Input.AsyncAPIFiles) == 0 &&
		len(cfg.Input.ServiceFiles) == 0 {
		return errors.New("at least one input source must be provided (dir, asyncapi_files, or service_files)")
	}

	return nil
}
