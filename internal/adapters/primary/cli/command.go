package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	docsgen "github.com/holydocs/holydocs/internal/adapters/secondary/docs"
	"github.com/holydocs/holydocs/internal/adapters/secondary/schema"
	d2target "github.com/holydocs/holydocs/internal/adapters/secondary/target/d2"
	"github.com/holydocs/holydocs/internal/config"
	"github.com/holydocs/holydocs/internal/core/app"
	"github.com/holydocs/messageflow/pkg/messageflow"
	mfschema "github.com/holydocs/messageflow/pkg/schema"
	mfd2 "github.com/holydocs/messageflow/pkg/schema/target/d2"
	do "github.com/samber/do/v2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// File permission.
const (
	dirPerm  = 0o755
	filePerm = 0o644
)

// Custom error types.
var (
	ErrNoSpecFilesProvided = errors.New("provide either asyncapi-files|servicefiles or dir")
	ErrNoSpecFilesFound    = errors.New("no specification files found in directory")
)

// Command represents the gen-docs command.
type Command struct {
	cmd           *cobra.Command
	app           *app.App
	schemaLoader  *schema.Loader
	docsGenerator *docsgen.Generator
	config        *config.Config
}

func NewCommand(i do.Injector) (*Command, error) {
	appInstance := do.MustInvoke[*app.App](i)
	schemaLoader := do.MustInvoke[*schema.Loader](i)
	docsGenerator := do.MustInvoke[*docsgen.Generator](i)
	cfg := do.MustInvoke[*config.Config](i)

	c := &Command{
		app:           appInstance,
		schemaLoader:  schemaLoader,
		docsGenerator: docsGenerator,
		config:        cfg,
	}

	c.cmd = &cobra.Command{
		Use:   "gen-docs",
		Short: "Generate system architecture documentation",
		Long: `Generate comprehensive documentation from ServiceFile and AsyncAPI specifications.

This command creates system architecture diagrams, service relationship maps, and 
message flow diagrams from your API specifications.

Input Sources:
  - Directory scanning: Automatically finds AsyncAPI and ServiceFile files
  - Specific files: Provide exact file paths for AsyncAPI and ServiceFile specs

Output:
  - D2 diagrams showing service relationships and message flows
  - README.md with system overview
  - JSON metadata

Examples:
  # Use configuration file
  holydocs gen-docs --config ./holydocs.yaml`,
		RunE: c.run,
	}

	return c, nil
}

// GetCommand returns the cobra command.
func (c *Command) GetCommand() *cobra.Command {
	return c.cmd
}

func (c *Command) run(_ *cobra.Command, _ []string) error {
	if err := c.prepareOutputDirectory(c.config.Output.Dir); err != nil {
		return fmt.Errorf("failed to prepare output directory: %w", err)
	}

	ctx := context.Background()

	if err := c.generateDocumentation(ctx, c.config); err != nil {
		return fmt.Errorf("failed to generate documentation: %w", err)
	}

	fmt.Printf("Documentation generated successfully in: %s\n", c.config.Output.Dir)

	return nil
}

func (c *Command) prepareOutputDirectory(outputDir string) error {
	if err := os.MkdirAll(outputDir, dirPerm); err != nil {
		return fmt.Errorf("creating output directory %s: %w", outputDir, err)
	}

	return nil
}

func (c *Command) generateDocumentation(ctx context.Context, cfg *config.Config) error {
	serviceFilesPaths, asyncAPIFilesPaths, err := c.getSpecFilesPaths(cfg)
	if err != nil {
		return fmt.Errorf("getting spec files paths: %w", err)
	}

	s, err := c.schemaLoader.Load(ctx, serviceFilesPaths, asyncAPIFilesPaths)
	if err != nil {
		return fmt.Errorf("loading schema from files: %w", err)
	}

	d2Target, err := d2target.NewTarget(cfg.Diagram.D2)
	if err != nil {
		return fmt.Errorf("creating D2 target: %w", err)
	}

	mfSetup, err := c.setupMessageFlowTarget(ctx, asyncAPIFilesPaths)
	if err != nil {
		return fmt.Errorf("setting up message flow target: %w", err)
	}

	newChangelog, err := c.docsGenerator.Generate(ctx, s, d2Target, mfSetup.Schema, mfSetup.Target, cfg)
	if err != nil {
		return fmt.Errorf("generating documentation: %w", err)
	}

	if newChangelog != nil && len(newChangelog.Changes) > 0 {
		fmt.Printf("\nNew Changes Detected:\n")
		for _, change := range newChangelog.Changes {
			fmt.Printf("• %s %s: %s\n", change.Type, change.Category, change.Details)
			if change.Diff != "" {
				fmt.Println(change.Diff)
			}
		}
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

func (c *Command) getSpecFilesPaths(cfg *config.Config) ([]string, []string, error) {
	if len(cfg.Input.ServiceFiles) != 0 || len(cfg.Input.AsyncAPIFiles) != 0 {
		return cfg.Input.ServiceFiles, cfg.Input.AsyncAPIFiles, nil
	}

	if cfg.Input.Dir != "" {
		return specFilesFromDir(cfg.Input.Dir)
	}

	return nil, nil, ErrNoSpecFilesProvided
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
			return nil //nolint:nilerr // Don't care about invalid YAML files
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
