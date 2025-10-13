package docs

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/holydocs/holydocs/pkg/config"
	"github.com/holydocs/holydocs/pkg/holydocs"
	"github.com/holydocs/holydocs/pkg/schema"
	d2target "github.com/holydocs/holydocs/pkg/schema/target/d2"
	mf "github.com/holydocs/messageflow/pkg/messageflow"
	mfschema "github.com/holydocs/messageflow/pkg/schema"
	mfd2 "github.com/holydocs/messageflow/pkg/schema/target/d2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateDocs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	asyncFiles, serviceFiles := getTestDataFiles()
	holydocsSchema, holydocsTarget, mfSchema, mfTarget := setupTestSchemasAndTargets(t, ctx, asyncFiles, serviceFiles)
	outputDir := filepath.Join(t.TempDir(), "docs")

	_, err := Generate(ctx, holydocsSchema, holydocsTarget, mfSchema, mfTarget, "HolyDOCs",
		"Internal Services", outputDir)
	if err != nil {
		t.Fatalf("generate docs: %v", err)
	}

	validateGeneratedFiles(t, outputDir)
}

func getTestDataFiles() ([]string, []string) {
	testdataDir := filepath.Join("..", "..", "pkg", "schema", "testdata")

	asyncFiles := []string{
		filepath.Join(testdataDir, "analytics.asyncapi.yaml"),
		filepath.Join(testdataDir, "campaign.asyncapi.yaml"),
		filepath.Join(testdataDir, "mailer.asyncapi.yaml"),
		filepath.Join(testdataDir, "notification.asyncapi.yaml"),
		filepath.Join(testdataDir, "reports.asyncapi.yaml"),
		filepath.Join(testdataDir, "user.asyncapi.yaml"),
	}

	serviceFiles := []string{
		filepath.Join(testdataDir, "analytics.servicefile.yml"),
		filepath.Join(testdataDir, "campaign.servicefile.yaml"),
		filepath.Join(testdataDir, "mailer.servicefile.yml"),
		filepath.Join(testdataDir, "notification.servicefile.yaml"),
		filepath.Join(testdataDir, "reports.servicefile.yml"),
		filepath.Join(testdataDir, "user.servicefile.yaml"),
	}

	return asyncFiles, serviceFiles
}

func setupTestSchemasAndTargets(t *testing.T, ctx context.Context, asyncFiles, serviceFiles []string) (
	holydocs.Schema, *d2target.Target, mf.Schema, *mfd2.Target) {
	holydocsSchema, err := schema.Load(ctx, serviceFiles, asyncFiles)
	if err != nil {
		t.Fatalf("load holydocs schema: %v", err)
	}

	holydocsTarget, err := d2target.NewTarget(config.D2Config{
		Pad:    64,
		Font:   "SourceSansPro",
		Layout: "elk",
	})
	if err != nil {
		t.Fatalf("create holydocs d2 target: %v", err)
	}

	mfSchema, err := mfschema.Load(ctx, asyncFiles)
	if err != nil {
		t.Fatalf("load messageflow schema: %v", err)
	}

	mfTarget, err := mfd2.NewTarget()
	if err != nil {
		t.Fatalf("create messageflow d2 target: %v", err)
	}

	return holydocsSchema, holydocsTarget, mfSchema, mfTarget
}

func TestProcessMetadata_FirstRun(t *testing.T) {
	tempDir := t.TempDir()

	schema := holydocs.Schema{
		Services: []holydocs.Service{
			{
				Info: holydocs.ServiceInfo{
					Name: "Test Service",
				},
			},
		},
	}

	metadata, newChangelog, err := processMetadata(schema, tempDir)

	require.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Nil(t, newChangelog, "Should not have changelog on first run")
	assert.Empty(t, metadata.Changelogs, "Should have empty changelogs on first run")
	assert.Equal(t, schema, metadata.Schema, "Should store the schema")
}

func TestProcessMetadata_SecondRunNoChanges(t *testing.T) {
	tempDir := t.TempDir()

	schema := holydocs.Schema{
		Services: []holydocs.Service{
			{
				Info: holydocs.ServiceInfo{
					Name: "Test Service",
				},
			},
		},
	}

	// First run
	_, _, err := processMetadata(schema, tempDir)
	require.NoError(t, err)

	// Second run with same schema
	metadata, newChangelog, err := processMetadata(schema, tempDir)

	require.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Nil(t, newChangelog, "Should not have changelog when no changes")
	assert.Empty(t, metadata.Changelogs, "Should still have empty changelogs")
}

func TestProcessMetadata_SecondRunWithChanges(t *testing.T) {
	tempDir := t.TempDir()

	oldSchema := holydocs.Schema{
		Services: []holydocs.Service{
			{
				Info: holydocs.ServiceInfo{
					Name: "Test Service",
				},
			},
		},
	}

	newSchema := holydocs.Schema{
		Services: []holydocs.Service{
			{
				Info: holydocs.ServiceInfo{
					Name: "Test Service",
				},
			},
			{
				Info: holydocs.ServiceInfo{
					Name: "New Service",
				},
			},
		},
	}

	// First run
	_, _, err := processMetadata(oldSchema, tempDir)
	require.NoError(t, err)

	// Second run with changes
	metadata, newChangelog, err := processMetadata(newSchema, tempDir)

	require.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.NotNil(t, newChangelog, "Should have changelog when changes detected")
	assert.Len(t, newChangelog.Changes, 1, "Should detect one change")
	assert.Equal(t, holydocs.ChangeTypeAdded, newChangelog.Changes[0].Type, "Should detect added service")
	assert.Len(t, metadata.Changelogs, 1, "Should have one changelog entry")
}

func TestReadMetadata_FileNotExists(t *testing.T) {
	tempDir := t.TempDir()

	metadata, err := readMetadata(tempDir)

	require.NoError(t, err)
	assert.Nil(t, metadata, "Should return nil when file doesn't exist")
}

func TestReadMetadata_FileExists(t *testing.T) {
	tempDir := t.TempDir()

	expectedMetadata := Metadata{
		Schema: holydocs.Schema{
			Services: []holydocs.Service{
				{
					Info: holydocs.ServiceInfo{
						Name: "Test Service",
					},
				},
			},
		},
		Changelogs: []holydocs.Changelog{
			{
				Date: time.Now(),
				Changes: []holydocs.Change{
					{
						Type:     holydocs.ChangeTypeAdded,
						Category: "service",
						Name:     "Test Service",
						Details:  "Test service was added",
					},
				},
			},
		},
	}

	err := writeMetadata(tempDir, expectedMetadata)
	require.NoError(t, err)

	metadata, err := readMetadata(tempDir)

	require.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, expectedMetadata.Schema, metadata.Schema)
	assert.Len(t, metadata.Changelogs, 1)
	assert.Equal(t, expectedMetadata.Changelogs[0].Changes[0].Type, metadata.Changelogs[0].Changes[0].Type)
}

func TestWriteMetadata(t *testing.T) {
	tempDir := t.TempDir()

	metadata := Metadata{
		Schema: holydocs.Schema{
			Services: []holydocs.Service{
				{
					Info: holydocs.ServiceInfo{
						Name: "Test Service",
					},
				},
			},
		},
		Changelogs: []holydocs.Changelog{},
	}

	err := writeMetadata(tempDir, metadata)

	require.NoError(t, err)

	// Verify file was created
	metadataPath := filepath.Join(tempDir, "holydocs.json")
	_, err = os.Stat(metadataPath)
	require.NoError(t, err, "Metadata file should be created")
}

func validateGeneratedFiles(t *testing.T, outputDir string) {
	expectedDir := filepath.Join("testdata", "expected")

	generatedFiles := collectFiles(t, outputDir)
	expectedFiles := collectFiles(t, expectedDir)

	if diff := cmp.Diff(sortedKeys(expectedFiles), sortedKeys(generatedFiles)); diff != "" {
		t.Fatalf("generated files mismatch expected (-want +got):\n%s", diff)
	}

	for path, expected := range expectedFiles {
		actual, ok := generatedFiles[path]
		if !ok {
			t.Fatalf("missing generated file: %s", path)
		}

		if !bytes.Equal(expected, actual) {
			validateFileContent(t, path, expected, actual)
		}
	}
}

func validateFileContent(t *testing.T, path string, expected, actual []byte) {
	if strings.HasSuffix(path, ".svg") {
		t.Fatalf("diagram %s does not match expected output", path)
	}
	if strings.HasSuffix(path, ".md") {
		if diff := cmp.Diff(string(expected), string(actual)); diff != "" {
			t.Fatalf("markdown %s mismatch (-want +got):\n%s", path, diff)
		}
	}
	t.Fatalf("file %s does not match expected output", path)
}

func collectFiles(t *testing.T, root string) map[string][]byte {
	files := make(map[string][]byte)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		files[filepath.ToSlash(rel)] = data

		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	return files
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}
