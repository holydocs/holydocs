package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Defaults(t *testing.T) {
	config, err := Load(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, "HolyDOCs", config.Output.Title)
	assert.Equal(t, "docs", config.Output.Directory)
	assert.Equal(t, "Internal Services", config.Output.GlobalName)
	assert.Equal(t, ".", config.Input.Directory)
}

func TestLoadConfig_FromYAML(t *testing.T) {
	// Create temporary YAML config file
	yamlContent := `
output:
  title: "YAML Test Title"
  directory: "/tmp/yaml-output"
  global_name: "YAML Global"
input:
  dir: "/tmp/yaml-input"
  asyncapi_files:
    - "/tmp/api1.asyncapi.yaml"
    - "/tmp/api2.asyncapi.yaml"
  service_files:
    - "/tmp/service1.servicefile.yaml"
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")
	err := os.WriteFile(configFile, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	config, err := Load(context.Background(), configFile)

	require.NoError(t, err)
	assert.Equal(t, "YAML Test Title", config.Output.Title)
	assert.Equal(t, "/tmp/yaml-output", config.Output.Directory)
	assert.Equal(t, "YAML Global", config.Output.GlobalName)
	assert.Equal(t, "/tmp/yaml-input", config.Input.Directory)
	assert.Equal(t, []string{"/tmp/api1.asyncapi.yaml", "/tmp/api2.asyncapi.yaml"}, config.Input.AsyncAPIFiles)
	assert.Equal(t, []string{"/tmp/service1.servicefile.yaml"}, config.Input.ServiceFiles)
}

func TestLoadConfig_FromEnv(t *testing.T) {
	// Create temporary YAML config file with specific values
	yamlContent := `
output:
  title: "YAML Title"
  directory: "/tmp/yaml-output"
  global_name: "YAML Global"
input:
  dir: "/tmp/yaml-input"
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")
	err := os.WriteFile(configFile, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	// Set environment variables with different values to test precedence
	// Based on aconfig documentation, environment variables should match struct field names
	envVars := map[string]string{
		"HOLYDOCS_OUTPUT_TITLE":       "ENV Title",
		"HOLYDOCS_OUTPUT_DIRECTORY":   "/tmp/env-output",
		"HOLYDOCS_OUTPUT_GLOBAL_NAME": "ENV Global",
		"HOLYDOCS_INPUT_DIRECTORY":    "/tmp/env-input",
	}

	// Set environment variables
	for key, value := range envVars {
		t.Setenv(key, value)
	}

	// Load configuration - environment variables should take precedence
	config, err := Load(context.Background(), configFile)

	require.NoError(t, err)

	// Assert that environment variables override YAML values
	assert.Equal(t, "ENV Title", config.Output.Title)
	assert.Equal(t, "/tmp/env-output", config.Output.Directory)
	assert.Equal(t, "ENV Global", config.Output.GlobalName)
	assert.Equal(t, "/tmp/env-input", config.Input.Directory)
}
