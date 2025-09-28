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

	"github.com/google/go-cmp/cmp"
	"github.com/holydocs/holydocs/pkg/holydocs"
	"github.com/holydocs/holydocs/pkg/schema"
	d2target "github.com/holydocs/holydocs/pkg/schema/target/d2"
	mf "github.com/holydocs/messageflow/pkg/messageflow"
	mfschema "github.com/holydocs/messageflow/pkg/schema"
	mfd2 "github.com/holydocs/messageflow/pkg/schema/target/d2"
)

func TestGenerateDocs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	asyncFiles, serviceFiles := getTestDataFiles()
	holydocsSchema, holydocsTarget, mfSchema, mfTarget := setupTestSchemasAndTargets(t, ctx, asyncFiles, serviceFiles)
	outputDir := filepath.Join(t.TempDir(), "docs")

	if err := Generate(ctx, holydocsSchema, holydocsTarget, mfSchema, mfTarget, "HolyDOCs",
		"Internal Services", outputDir); err != nil {
		t.Fatalf("generate docs: %v", err)
	}

	validateGeneratedFiles(t, outputDir)
}

func getTestDataFiles() ([]string, []string) {
	testdataDir := filepath.Join("..", "..", "pkg", "schema", "testdata")

	asyncFiles := []string{
		filepath.Join(testdataDir, "analytics.asyncapi.yaml"),
		filepath.Join(testdataDir, "campaign.analytics.yaml"),
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

	holydocsTarget, err := d2target.NewTarget()
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
