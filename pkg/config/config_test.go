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
	assert.Equal(t, "docs", config.Output.Dir)
	assert.Equal(t, "Internal Services", config.Output.GlobalName)
	assert.Equal(t, ".", config.Input.Dir)

	assert.Equal(t, int64(64), config.Diagram.D2.Pad)
	assert.Equal(t, int64(0), config.Diagram.D2.Theme)
	assert.False(t, config.Diagram.D2.Sketch)
	assert.Equal(t, "SourceSansPro", config.Diagram.D2.Font)
	assert.Equal(t, "elk", config.Diagram.D2.Layout)
}

func TestLoadConfig_FromYAML(t *testing.T) {
	// Create temporary YAML config file
	yamlContent := `
output:
  title: "YAML Test Title"
  dir: "/tmp/yaml-output"
  global_name: "YAML Global"
input:
  dir: "/tmp/yaml-input"
  asyncapi_files:
    - "/tmp/api1.asyncapi.yaml"
    - "/tmp/api2.asyncapi.yaml"
  service_files:
    - "/tmp/service1.servicefile.yaml"
diagram:
  d2:
    pad: 100
    theme: 1
    sketch: true
    font: "SourceCodePro"
    layout: "elk"
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")
	err := os.WriteFile(configFile, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	config, err := Load(context.Background(), configFile)

	require.NoError(t, err)
	assert.Equal(t, "YAML Test Title", config.Output.Title)
	assert.Equal(t, "/tmp/yaml-output", config.Output.Dir)
	assert.Equal(t, "YAML Global", config.Output.GlobalName)
	assert.Equal(t, "/tmp/yaml-input", config.Input.Dir)
	assert.Equal(t, []string{"/tmp/api1.asyncapi.yaml", "/tmp/api2.asyncapi.yaml"}, config.Input.AsyncAPIFiles)
	assert.Equal(t, []string{"/tmp/service1.servicefile.yaml"}, config.Input.ServiceFiles)

	// Test Diagram.D2 configuration from YAML
	assert.Equal(t, int64(100), config.Diagram.D2.Pad)
	assert.Equal(t, int64(1), config.Diagram.D2.Theme)
	assert.True(t, config.Diagram.D2.Sketch)
	assert.Equal(t, "SourceCodePro", config.Diagram.D2.Font)
	assert.Equal(t, "elk", config.Diagram.D2.Layout)
}

func TestLoadConfig_FromEnv(t *testing.T) {
	// Create temporary YAML config file with specific values
	yamlContent := `
output:
  title: "YAML Title"
  dir: "/tmp/yaml-output"
  global_name: "YAML Global"
input:
  dir: "/tmp/yaml-input"
diagram:
  d2:
    pad: 100
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")
	err := os.WriteFile(configFile, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	// Set environment variables with different values to test precedence
	// Based on aconfig documentation, environment variables should match struct field names
	envVars := map[string]string{
		"HOLYDOCS_OUTPUT_TITLE":       "ENV Title",
		"HOLYDOCS_OUTPUT_DIR":         "/tmp/env-output",
		"HOLYDOCS_OUTPUT_GLOBAL_NAME": "ENV Global",
		"HOLYDOCS_INPUT_DIR":          "/tmp/env-input",
		"HOLYDOCS_DIAGRAM_D2_PAD":     "200",
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
	assert.Equal(t, "/tmp/env-output", config.Output.Dir)
	assert.Equal(t, "ENV Global", config.Output.GlobalName)
	assert.Equal(t, "/tmp/env-input", config.Input.Dir)
	assert.Equal(t, int64(200), config.Diagram.D2.Pad)
}
