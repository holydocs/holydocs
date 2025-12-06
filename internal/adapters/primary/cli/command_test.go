package cli

import (
	"os"
	"path/filepath"
	"testing"

	docsgen "github.com/holydocs/holydocs/internal/adapters/secondary/docs"
	"github.com/holydocs/holydocs/internal/adapters/secondary/schema"
	"github.com/holydocs/holydocs/internal/config"
	"github.com/holydocs/holydocs/internal/core/app"
	do "github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestInjector() do.Injector {
	injector := do.New()
	do.Provide(injector, func(i do.Injector) (*app.App, error) {
		return app.NewApp(), nil
	})
	do.Provide(injector, schema.NewLoader)
	do.Provide(injector, docsgen.NewGenerator)
	do.ProvideValue(injector, config.ConfigFilePath(""))
	do.Provide(injector, config.LoadConfig)

	return injector
}

func TestNewCommand(t *testing.T) {
	t.Parallel()

	injector := setupTestInjector()
	cmd, err := NewCommand(injector)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.NotNil(t, cmd.cmd)
	assert.Equal(t, "gen-docs", cmd.cmd.Use)
}

func TestCommand_GetCommand(t *testing.T) {
	t.Parallel()

	injector := setupTestInjector()
	cmd, err := NewCommand(injector)
	require.NoError(t, err)
	cobraCmd := cmd.GetCommand()
	require.NotNil(t, cobraCmd)
	assert.Equal(t, cmd.cmd, cobraCmd)
}

func TestCommand_prepareOutputDirectory(t *testing.T) {
	t.Parallel()

	injector := setupTestInjector()
	cmd, err := NewCommand(injector)
	require.NoError(t, err)

	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "output")

	err = cmd.prepareOutputDirectory(outputDir)
	require.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(outputDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestCommand_getSpecFilesPaths_FromConfig(t *testing.T) {
	t.Parallel()

	injector := setupTestInjector()
	cmd, err := NewCommand(injector)
	require.NoError(t, err)

	cfg := &config.Config{
		Input: config.Input{
			ServiceFiles:  []string{"service1.yaml", "service2.yaml"},
			AsyncAPIFiles: []string{"async1.yaml", "async2.yaml"},
		},
	}

	serviceFiles, asyncFiles, err := cmd.getSpecFilesPaths(cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{"service1.yaml", "service2.yaml"}, serviceFiles)
	assert.Equal(t, []string{"async1.yaml", "async2.yaml"}, asyncFiles)
}

func TestCommand_getSpecFilesPaths_FromDir(t *testing.T) {
	t.Parallel()

	injector := setupTestInjector()
	cmd, err := NewCommand(injector)
	require.NoError(t, err)

	tempDir := t.TempDir()

	// Create test files
	asyncAPIFile := filepath.Join(tempDir, "test.asyncapi.yaml")
	serviceFile := filepath.Join(tempDir, "test.servicefile.yaml")

	// Write AsyncAPI file
	err = os.WriteFile(asyncAPIFile, []byte(`asyncapi: "2.0.0"
info:
  title: Test API
`), 0o644)
	require.NoError(t, err)

	// Write ServiceFile
	err = os.WriteFile(serviceFile, []byte(`servicefile: "1.0.0"
info:
  name: Test Service
`), 0o644)
	require.NoError(t, err)

	cfg := &config.Config{
		Input: config.Input{
			Dir: tempDir,
		},
	}

	serviceFiles, asyncFiles, err := cmd.getSpecFilesPaths(cfg)
	require.NoError(t, err)
	assert.Contains(t, serviceFiles, serviceFile)
	assert.Contains(t, asyncFiles, asyncAPIFile)
}

func TestCommand_getSpecFilesPaths_NoFilesProvided(t *testing.T) {
	t.Parallel()

	injector := setupTestInjector()
	cmd, err := NewCommand(injector)
	require.NoError(t, err)

	cfg := &config.Config{
		Input: config.Input{},
	}

	serviceFiles, asyncFiles, err := cmd.getSpecFilesPaths(cfg)
	require.Error(t, err)
	assert.Equal(t, ErrNoSpecFilesProvided, err)
	assert.Nil(t, serviceFiles)
	assert.Nil(t, asyncFiles)
}

func TestSpecFilesFromDir_EmptyDirectory(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	serviceFiles, asyncFiles, err := specFilesFromDir(tempDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrNoSpecFilesFound.Error())
	assert.Nil(t, serviceFiles)
	assert.Nil(t, asyncFiles)
}

func TestSpecFilesFromDir_WithAsyncAPI(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	asyncAPIFile := filepath.Join(tempDir, "test.asyncapi.yaml")
	err := os.WriteFile(asyncAPIFile, []byte(`asyncapi: "2.0.0"
info:
  title: Test API
`), 0o644)
	require.NoError(t, err)

	serviceFiles, asyncFiles, err := specFilesFromDir(tempDir)
	require.NoError(t, err)
	assert.Empty(t, serviceFiles)
	assert.Contains(t, asyncFiles, asyncAPIFile)
}

func TestSpecFilesFromDir_WithServiceFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	serviceFile := filepath.Join(tempDir, "test.servicefile.yaml")
	err := os.WriteFile(serviceFile, []byte(`servicefile: "1.0.0"
info:
  name: Test Service
`), 0o644)
	require.NoError(t, err)

	serviceFiles, asyncFiles, err := specFilesFromDir(tempDir)
	require.NoError(t, err)
	assert.Contains(t, serviceFiles, serviceFile)
	assert.Empty(t, asyncFiles)
}

func TestSpecFilesFromDir_WithBoth(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	asyncAPIFile := filepath.Join(tempDir, "test.asyncapi.yaml")
	err := os.WriteFile(asyncAPIFile, []byte(`asyncapi: "2.0.0"
info:
  title: Test API
`), 0o644)
	require.NoError(t, err)

	serviceFile := filepath.Join(tempDir, "test.servicefile.yaml")
	err = os.WriteFile(serviceFile, []byte(`servicefile: "1.0.0"
info:
  name: Test Service
`), 0o644)
	require.NoError(t, err)

	serviceFiles, asyncFiles, err := specFilesFromDir(tempDir)
	require.NoError(t, err)
	assert.Contains(t, serviceFiles, serviceFile)
	assert.Contains(t, asyncFiles, asyncAPIFile)
}

func TestSpecFilesFromDir_IgnoresNonYAMLFiles(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create a non-YAML file
	textFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(textFile, []byte("not yaml"), 0o644)
	require.NoError(t, err)

	// Non-YAML files are ignored, so we should get an error for no spec files found
	serviceFiles, asyncFiles, err := specFilesFromDir(tempDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrNoSpecFilesFound.Error())
	assert.Nil(t, serviceFiles)
	assert.Nil(t, asyncFiles)
}

func TestSpecFilesFromDir_IgnoresInvalidYAML(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create invalid YAML file (invalid YAML is silently ignored by the function)
	invalidFile := filepath.Join(tempDir, "invalid.yaml")
	err := os.WriteFile(invalidFile, []byte("not valid yaml: [["), 0o644)
	require.NoError(t, err)

	// Invalid YAML files are silently ignored, so we should get an error for no spec files found
	serviceFiles, asyncFiles, err := specFilesFromDir(tempDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrNoSpecFilesFound.Error())
	assert.Nil(t, serviceFiles)
	assert.Nil(t, asyncFiles)
}

func TestSpecFilesFromDir_Recursive(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	err := os.MkdirAll(subDir, 0o755)
	require.NoError(t, err)

	asyncAPIFile := filepath.Join(subDir, "test.asyncapi.yaml")
	err = os.WriteFile(asyncAPIFile, []byte(`asyncapi: "2.0.0"
info:
  title: Test API
`), 0o644)
	require.NoError(t, err)

	_, asyncFiles, err := specFilesFromDir(tempDir)
	require.NoError(t, err)
	assert.Contains(t, asyncFiles, asyncAPIFile)
}

func TestMapKeysSorted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    map[string]struct{}
		expected []string
	}{
		{
			name:     "empty map",
			input:    map[string]struct{}{},
			expected: nil,
		},
		{
			name: "single key",
			input: map[string]struct{}{
				"a": {},
			},
			expected: []string{"a"},
		},
		{
			name: "multiple keys",
			input: map[string]struct{}{
				"c": {},
				"a": {},
				"b": {},
			},
			expected: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapKeysSorted(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
