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
	Input         Input         `env:"INPUT" yaml:"input"`
	Output        Output        `env:"OUTPUT" yaml:"output"`
	Diagram       Diagram       `env:"DIAGRAM" yaml:"diagram"`
	Documentation Documentation `env:"DOCUMENTATION" yaml:"documentation"`
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

// Markdown represents markdown content that can be sourced from either a string or a file.
type Markdown struct {
	Content  string `env:"CONTENT" yaml:"content" usage:"Raw markdown content"`
	FilePath string `env:"FILE_PATH" yaml:"filePath" usage:"Path to a markdown file"`
}

// Documentation represents documentation configuration for extending generated docs with custom markdown.
type Documentation struct {
	Overview OverviewDocumentation           `env:"OVERVIEW" yaml:"overview" usage:"Markdown content to place after overview diagram"`
	Services map[string]ServiceDocumentation `env:"SERVICES" yaml:"services" usage:"Markdown content for specific services to place after service relationship diagrams"`
	Systems  map[string]SystemDocumentation  `env:"SYSTEMS" yaml:"systems" usage:"Markdown content for specific systems to place after system diagrams"`
}

type OverviewDocumentation struct {
	Description Markdown `env:"DESCRIPTION" yaml:"description" usage:"Markdown content to place after overview diagram"`
}

type ServiceDocumentation struct {
	Summary     Markdown `env:"SUMMARY" yaml:"summary" usage:"Summary of the service"`
	Description Markdown `env:"DESCRIPTION" yaml:"description" usage:"Markdown content for specific services to place after service relationship diagrams"`
}

type SystemDocumentation struct {
	Summary     Markdown `env:"SUMMARY" yaml:"summary" usage:"Summary of the system"`
	Description Markdown `env:"DESCRIPTION" yaml:"description" usage:"Markdown content for specific services to place after service relationship diagrams"`
}

type SystemsDocumentation struct {
	Description Markdown `env:"DESCRIPTION" yaml:"description" usage:"Markdown content for specific systems to place after system diagrams"`
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

	// Validate documentation configuration
	if err := validateDocumentation(&cfg.Documentation); err != nil {
		return fmt.Errorf("invalid documentation configuration: %w", err)
	}

	return nil
}

// validateDocumentation validates the documentation configuration.
func validateDocumentation(doc *Documentation) error {
	// Validate overview markdown
	if err := validateMarkdown(&doc.Overview.Description, "overview description"); err != nil {
		return err
	}

	// Validate services markdown
	for serviceName, serviceDoc := range doc.Services {
		if err := validateMarkdown(&serviceDoc.Summary, "service "+serviceName+" summary"); err != nil {
			return err
		}
		if err := validateMarkdown(&serviceDoc.Description, "service "+serviceName+" description"); err != nil {
			return err
		}
	}

	// Validate systems markdown
	for systemName, systemDoc := range doc.Systems {
		if err := validateMarkdown(&systemDoc.Summary, "system "+systemName+" summary"); err != nil {
			return err
		}
		if err := validateMarkdown(&systemDoc.Description, "system "+systemName+" description"); err != nil {
			return err
		}
	}

	return nil
}

// validateMarkdown validates a single markdown configuration.
func validateMarkdown(md *Markdown, context string) error {
	// Check that either content or file path is provided, but not both
	hasContent := md.Content != ""
	hasFilePath := md.FilePath != ""

	if hasContent && hasFilePath {
		return fmt.Errorf("%s: cannot specify both content and file_path", context)
	}

	// At least one should be provided if the markdown is being used
	// Note: We don't require either to be set since markdown sections are optional
	return nil
}
