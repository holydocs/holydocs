package docs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/holydocs/holydocs/internal/docs"
	"github.com/holydocs/holydocs/pkg/schema"
	"github.com/holydocs/holydocs/pkg/schema/target/d2"
	"github.com/holydocs/messageflow/pkg/messageflow"
	mfschema "github.com/holydocs/messageflow/pkg/schema"
	mfd2 "github.com/holydocs/messageflow/pkg/schema/target/d2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// File permission.
const (
	dirPerm  = 0o755
	filePerm = 0o644
)

// Command flag.
const (
	flagDir           = "dir"
	flagAsyncAPIFiles = "asyncapi-files"
	flagServiceFiles  = "servicefiles"
	flagOutput        = "output"
	flagTitle         = "title"
	flagGlobalName    = "global-name"
)

// Default values.
const (
	defaultOutputDir = "."
	defaultTitle     = "HolyDOCs"
)

// Custom error types.
var (
	ErrNoSpecFilesProvided = errors.New("provide either asyncapi-files|servicefiles or dir")
	ErrNoSpecFilesFound    = errors.New("no specification files found in directory")
	ErrInvalidYAMLFile     = errors.New("invalid YAML file")
)

// Command represents the gen-docs command.
type Command struct {
	cmd *cobra.Command
}

// NewCommand creates a new gen-docs command.
func NewCommand() *Command {
	c := &Command{}

	c.cmd = &cobra.Command{
		Use:   "gen-docs",
		Short: "Generate documentation",
		Long: `Generate comprehensive documentation from ServiceFile/AsyncAPI.
Example:
  holydocs gen --dir ./specs --output ./docs`,
		RunE: c.run,
	}

	c.setupFlags()

	return c
}

// setupFlags configures the command flags.
func (c *Command) setupFlags() {
	c.cmd.Flags().String(flagDir, "", "Path to dir to scan spec files automatically")
	c.cmd.Flags().String(flagAsyncAPIFiles, "", "Paths to asyncapi files separated by comma")
	c.cmd.Flags().String(flagServiceFiles, "", "Paths to servicefiles separated by comma")
	c.cmd.Flags().String(flagOutput, defaultOutputDir, "Output directory for generated documentation")
	c.cmd.Flags().String(flagTitle, defaultTitle, "Title of the documentation")
	c.cmd.Flags().String(flagGlobalName, "", "Custom name for internal services container (default: 'Internal Services')")
}

// GetCommand returns the cobra command.
func (c *Command) GetCommand() *cobra.Command {
	return c.cmd
}

func (c *Command) run(cmd *cobra.Command, _ []string) error {
	config, err := c.parseFlags(cmd)
	if err != nil {
		return fmt.Errorf("failed to parse command flags: %w", err)
	}

	if err := c.validateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	if err := c.prepareOutputDirectory(config.OutputDir); err != nil {
		return fmt.Errorf("failed to prepare output directory: %w", err)
	}

	ctx := context.Background()

	if err := c.generateDocumentation(ctx, config); err != nil {
		return fmt.Errorf("failed to generate documentation: %w", err)
	}

	fmt.Printf("Documentation generated successfully in: %s\n", config.OutputDir)

	return nil
}

// CommandConfig holds the parsed command configuration.
type CommandConfig struct {
	Title              string
	GlobalName         string
	ServiceFilesPaths  []string
	AsyncAPIFilesPaths []string
	OutputDir          string
}

// parseFlags extracts and validates command flags.
func (c *Command) parseFlags(cmd *cobra.Command) (*CommandConfig, error) {
	title, err := cmd.Flags().GetString(flagTitle)
	if err != nil {
		return nil, fmt.Errorf("getting title flag: %w", err)
	}

	globalName, err := cmd.Flags().GetString(flagGlobalName)
	if err != nil {
		return nil, fmt.Errorf("getting global-name flag: %w", err)
	}

	outputDir, err := cmd.Flags().GetString(flagOutput)
	if err != nil {
		return nil, fmt.Errorf("getting output flag: %w", err)
	}

	serviceFilesPaths, asyncAPIFilesPaths, err := getSpecFilesPaths(cmd)
	if err != nil {
		return nil, fmt.Errorf("getting spec files paths: %w", err)
	}

	return &CommandConfig{
		Title:              title,
		GlobalName:         globalName,
		ServiceFilesPaths:  serviceFilesPaths,
		AsyncAPIFilesPaths: asyncAPIFilesPaths,
		OutputDir:          outputDir,
	}, nil
}

// validateConfig validates the command configuration.
func (c *Command) validateConfig(config *CommandConfig) error {
	if config.Title == "" {
		return errors.New("title cannot be empty")
	}

	if config.OutputDir == "" {
		return errors.New("output directory cannot be empty")
	}

	return nil
}

// prepareOutputDirectory creates the output directory if it doesn't exist.
func (c *Command) prepareOutputDirectory(outputDir string) error {
	if err := os.MkdirAll(outputDir, dirPerm); err != nil {
		return fmt.Errorf("creating output directory %s: %w", outputDir, err)
	}

	return nil
}

// generateDocumentation generates the documentation using the provided configuration.
func (c *Command) generateDocumentation(ctx context.Context, config *CommandConfig) error {
	s, err := schema.Load(ctx, config.ServiceFilesPaths, config.AsyncAPIFilesPaths)
	if err != nil {
		return fmt.Errorf("loading schema from files: %w", err)
	}

	d2Target, err := d2.NewTarget()
	if err != nil {
		return fmt.Errorf("creating D2 target: %w", err)
	}

	mfSetup, err := c.setupMessageFlowTarget(ctx, config.AsyncAPIFilesPaths)
	if err != nil {
		return fmt.Errorf("setting up message flow target: %w", err)
	}

	if err := docs.Generate(ctx, s, d2Target, mfSetup.Schema, mfSetup.Target, config.Title, config.GlobalName,
		config.OutputDir); err != nil {
		return fmt.Errorf("generating documentation: %w", err)
	}

	return nil
}

// MessageFlowSetup holds the message flow schema and target.
type MessageFlowSetup struct {
	Schema messageflow.Schema
	Target messageflow.Target
}

// setupMessageFlowTarget sets up the message flow target if AsyncAPI files are provided.
func (c *Command) setupMessageFlowTarget(ctx context.Context, asyncAPIFilesPaths []string) (*MessageFlowSetup, error) {
	if len(asyncAPIFilesPaths) == 0 {
		return &MessageFlowSetup{}, nil
	}

	mfSchema, err := mfschema.Load(ctx, asyncAPIFilesPaths)
	if err != nil {
		return nil, fmt.Errorf("loading messageflow schema: %w", err)
	}

	mfTarget, err := mfd2.NewTarget()
	if err != nil {
		return nil, fmt.Errorf("creating messageflow D2 target: %w", err)
	}

	return &MessageFlowSetup{
		Schema: mfSchema,
		Target: mfTarget,
	}, nil
}

func getSpecFilesPaths(cmd *cobra.Command) ([]string, []string, error) {
	specDir, err := cmd.Flags().GetString(flagDir)
	if err != nil {
		return nil, nil, fmt.Errorf("getting dir flag: %w", err)
	}

	if specDir != "" {
		return specFilesFromDir(specDir)
	}

	asyncAPIFilesPathsStr, err := cmd.Flags().GetString(flagAsyncAPIFiles)
	if err != nil {
		return nil, nil, fmt.Errorf("getting asyncapi-files flag: %w", err)
	}

	serviceFilesPathsStr, err := cmd.Flags().GetString(flagServiceFiles)
	if err != nil {
		return nil, nil, fmt.Errorf("getting servicefiles flag: %w", err)
	}

	asyncPaths := splitAndCleanPaths(asyncAPIFilesPathsStr)
	servicePaths := splitAndCleanPaths(serviceFilesPathsStr)

	if len(asyncPaths) == 0 && len(servicePaths) == 0 {
		return nil, nil, ErrNoSpecFilesProvided
	}

	return servicePaths, asyncPaths, nil
}

func splitAndCleanPaths(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	items := strings.Split(value, ",")
	clean := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		clean = append(clean, trimmed)
	}

	sort.Strings(clean)

	return clean
}

func specFilesFromDir(dir string) ([]string, []string, error) {
	fmt.Println("Scanning directory for spec files:", dir)

	asyncMap := make(map[string]struct{})
	serviceMap := make(map[string]struct{})

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading file %s: %w", path, err)
		}

		var yamlDoc map[string]interface{}
		if err := yaml.Unmarshal(content, &yamlDoc); err != nil {
			return fmt.Errorf("%w %s: %w", ErrInvalidYAMLFile, path, err)
		}

		if _, hasAsyncAPI := yamlDoc["asyncapi"]; hasAsyncAPI {
			asyncMap[path] = struct{}{}
		}

		if _, hasServiceFile := yamlDoc["servicefile"]; hasServiceFile {
			serviceMap[path] = struct{}{}
		}

		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("error walking directory %s: %w", dir, err)
	}

	asyncAPIFiles := mapKeysSorted(asyncMap)
	serviceFiles := mapKeysSorted(serviceMap)

	if len(asyncAPIFiles) == 0 && len(serviceFiles) == 0 {
		return nil, nil, fmt.Errorf("%w in directory %s", ErrNoSpecFilesFound, dir)
	}

	fmt.Println("Found AsyncAPI files:", asyncAPIFiles)
	fmt.Println("Found ServiceFile files:", serviceFiles)

	return serviceFiles, asyncAPIFiles, nil
}

func mapKeysSorted(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}
