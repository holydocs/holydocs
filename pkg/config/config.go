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

// Config represents the complete configuration for HolyDOCs.
type Config struct {
	Input   Input   `env:"INPUT" yaml:"input"`
	Output  Output  `env:"OUTPUT" yaml:"output"`
	Diagram Diagram `env:"DIAGRAM" yaml:"diagram"`
}

// Input represents input configuration for HolyDOCs.
type Input struct {
	Dir           string   `env:"DIR" yaml:"dir" default:"." usage:"Directory to scan for AsyncAPI and ServiceFile files"`
	AsyncAPIFiles []string `env:"ASYNCAPI_FILES" yaml:"asyncapi_files" usage:"Comma-separated list of AsyncAPI specification files"`
	ServiceFiles  []string `env:"SERVICE_FILES" yaml:"service_files" usage:"Comma-separated list of ServiceFile specification files"`
}

// Output represents output configuration for HolyDOCs.
type Output struct {
	Dir        string `env:"DIR" yaml:"dir" default:"docs" usage:"Directory where documentation will be generated"`
	Title      string `env:"TITLE" yaml:"title" default:"HolyDOCs" usage:"Title for the generated documentation"`
	GlobalName string `env:"GLOBAL_NAME" yaml:"global_name" default:"Internal Services" usage:"Name used for grouping internal services in diagrams"`
}

// Diagram represents diagram generation configuration for HolyDOCs.
type Diagram struct {
	D2 D2Config `env:"D2" yaml:"d2"`
}

// D2Config represents D2 diagram generation configuration.
type D2Config struct {
	// Render settings
	Pad    int64 `env:"PAD" yaml:"pad" default:"64" usage:"Padding around the diagram in pixels"`
	Theme  int64 `env:"THEME" yaml:"theme" default:"0" usage:"Theme ID for the diagram (0 for default, -1 for dark)"`
	Sketch bool  `env:"SKETCH" yaml:"sketch" default:"false" usage:"Enable sketch mode for hand-drawn appearance"`

	// Font and layout settings
	Font   string `env:"FONT" yaml:"font" default:"SourceSansPro" usage:"Font family for diagram text (SourceSansPro, SourceCodePro, HandDrawn)"`
	Layout string `env:"LAYOUT" yaml:"layout" default:"elk" usage:"Layout engine for diagram arrangement (dagre, elk)"`
}

// Load loads configuration from multiple sources in priority order:
// 1. Environment variables
// 2. YAML configuration file.
func Load(_ context.Context, configFile string) (*Config, error) {
	loaderConfig := aconfig.Config{
		EnvPrefix: "HOLYDOCS",
		SkipFlags: true,
		FileDecoders: map[string]aconfig.FileDecoder{
			".yaml": aconfigyaml.New(),
			".yml":  aconfigyaml.New(),
		},
	}

	if configFile != "" {
		loaderConfig.Files = []string{configFile}
	}

	cfg := &Config{}

	if err := aconfig.LoaderFor(cfg, loaderConfig).Load(); err != nil {
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

	if cfg.Output.Dir == "" {
		return errors.New("output directory cannot be empty")
	}

	// Validate that at least one input source is provided
	if cfg.Input.Dir == "" &&
		len(cfg.Input.AsyncAPIFiles) == 0 &&
		len(cfg.Input.ServiceFiles) == 0 {
		return errors.New("at least one input source must be provided (dir, asyncapi_files, or service_files)")
	}

	return nil
}
