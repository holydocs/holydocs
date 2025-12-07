package d2

import (
	"context"
	"testing"

	"github.com/holydocs/holydocs/internal/config"
	"github.com/holydocs/holydocs/internal/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTarget(t *testing.T) {
	t.Parallel()

	tests := getNewTargetTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, err := NewTarget(tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, target)
			} else {
				require.NoError(t, err)
				require.NotNil(t, target)
				assert.NotNil(t, target.overviewTemplate)
				assert.NotNil(t, target.serviceRelationshipsTemplate)
				assert.NotNil(t, target.systemTemplate)
			}
		})
	}
}

func getNewTargetTestCases() []struct {
	name    string
	cfg     config.D2Config
	wantErr bool
} {
	return []struct {
		name    string
		cfg     config.D2Config
		wantErr bool
	}{
		{
			name: "valid default config",
			cfg: config.D2Config{
				Pad:    64,
				Font:   "SourceSansPro",
				Layout: "elk",
			},
			wantErr: false,
		},
		{
			name: "valid config with theme",
			cfg: config.D2Config{
				Pad:    64,
				Font:   "SourceSansPro",
				Layout: "elk",
				Theme:  1,
			},
			wantErr: false,
		},
		{
			name: "valid config with sketch",
			cfg: config.D2Config{
				Pad:    64,
				Font:   "SourceSansPro",
				Layout: "elk",
				Sketch: true,
			},
			wantErr: false,
		},
		{
			name: "valid config with dagre layout",
			cfg: config.D2Config{
				Pad:    64,
				Font:   "SourceSansPro",
				Layout: "dagre",
			},
			wantErr: false,
		},
	}
}

func TestTarget_Capabilities(t *testing.T) {
	t.Parallel()

	cfg := config.D2Config{
		Pad:    64,
		Font:   "SourceSansPro",
		Layout: "elk",
	}
	target, err := NewTarget(cfg)
	require.NoError(t, err)

	caps := target.Capabilities()
	assert.True(t, caps.Format)
	assert.True(t, caps.Render)
}

func TestTarget_FormatSchema(t *testing.T) {
	t.Parallel()

	cfg := config.D2Config{
		Pad:    64,
		Font:   "SourceSansPro",
		Layout: "elk",
	}
	target, err := NewTarget(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	schema := domain.Schema{}
	opts := domain.FormatOptions{}

	result, err := target.FormatSchema(ctx, schema, opts)
	require.Error(t, err)
	assert.Equal(t, ErrFormatSchemaNotSupported, err)
	assert.Equal(t, domain.FormattedSchema{}, result)
}

func TestTarget_RenderSchema_NilContext(t *testing.T) {
	t.Parallel()

	cfg := config.D2Config{
		Pad:    64,
		Font:   "SourceSansPro",
		Layout: "elk",
	}
	target, err := NewTarget(cfg)
	require.NoError(t, err)

	fs := domain.FormattedSchema{
		Type: domain.TargetType("d2"),
		Data: []byte("x -> y"),
	}

	//nolint:staticcheck // Intentionally testing nil context rejection
	result, err := target.RenderSchema(nil, fs)
	require.Error(t, err)
	assert.Equal(t, ErrContextRequired, err)
	assert.Nil(t, result)
}

func TestTarget_RenderSchema_WrongType(t *testing.T) {
	t.Parallel()

	cfg := config.D2Config{
		Pad:    64,
		Font:   "SourceSansPro",
		Layout: "elk",
	}
	target, err := NewTarget(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	fs := domain.FormattedSchema{
		Type: domain.TargetType("invalid"),
		Data: []byte("x -> y"),
	}

	result, err := target.RenderSchema(ctx, fs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format type")
	assert.Nil(t, result)
}

func TestTarget_RenderSchema_ValidD2(t *testing.T) {
	t.Parallel()

	cfg := config.D2Config{
		Pad:    64,
		Font:   "SourceSansPro",
		Layout: "elk",
	}
	target, err := NewTarget(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	fs := domain.FormattedSchema{
		Type: domain.TargetType("d2"),
		Data: []byte("x -> y"),
	}

	result, err := target.RenderSchema(ctx, fs)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, string(result), "<svg")
}

func TestTarget_RenderSchema_InvalidD2Syntax(t *testing.T) {
	t.Parallel()

	cfg := config.D2Config{
		Pad:    64,
		Font:   "SourceSansPro",
		Layout: "elk",
	}
	target, err := NewTarget(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	fs := domain.FormattedSchema{
		Type: domain.TargetType("d2"),
		Data: []byte("invalid d2 syntax {{{{"),
	}

	result, err := target.RenderSchema(ctx, fs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compile diagram")
	assert.Nil(t, result)
}

func TestTarget_GenerateOverviewDiagramScript(t *testing.T) {
	t.Parallel()

	cfg := config.D2Config{
		Pad:    64,
		Font:   "SourceSansPro",
		Layout: "elk",
	}
	target, err := NewTarget(cfg)
	require.NoError(t, err)

	schema := domain.Schema{
		Services: []domain.Service{
			{
				Info: domain.ServiceInfo{
					Name: "Test Service",
				},
			},
		},
	}
	asyncEdges := []domain.AsyncEdge{}

	script, err := target.GenerateOverviewDiagramScript(schema, asyncEdges, "Test System")
	require.NoError(t, err)
	assert.NotEmpty(t, script)
	assert.Contains(t, string(script), "Test Service")
}

func TestTarget_GenerateServiceRelationshipsDiagramScript(t *testing.T) {
	t.Parallel()

	cfg := config.D2Config{
		Pad:    64,
		Font:   "SourceSansPro",
		Layout: "elk",
	}
	target, err := NewTarget(cfg)
	require.NoError(t, err)

	service := domain.Service{
		Info: domain.ServiceInfo{
			Name: "Test Service",
		},
		Relationships: []domain.Relationship{
			{
				Action:      domain.RelationshipActionUses,
				Participant: "Database",
				Technology:  "PostgreSQL",
			},
		},
	}
	allServices := []domain.Service{service}
	asyncEdges := []domain.AsyncEdge{}

	script, err := target.GenerateServiceRelationshipsDiagramScript(service, allServices, asyncEdges)
	require.NoError(t, err)
	assert.NotEmpty(t, script)
	assert.Contains(t, string(script), "Test Service")
}

func TestTarget_GenerateSystemDiagramScript(t *testing.T) {
	t.Parallel()

	cfg := config.D2Config{
		Pad:    64,
		Font:   "SourceSansPro",
		Layout: "elk",
	}
	target, err := NewTarget(cfg)
	require.NoError(t, err)

	schema := domain.Schema{
		Services: []domain.Service{
			{
				Info: domain.ServiceInfo{
					Name:   "Test Service",
					System: "Test System",
				},
			},
		},
	}
	asyncEdges := []domain.AsyncEdge{}

	script, err := target.GenerateSystemDiagramScript(schema, "Test System", asyncEdges)
	require.NoError(t, err)
	assert.NotEmpty(t, script)
	assert.Contains(t, string(script), "Test System")
}

func TestTarget_GenerateOverviewDiagram(t *testing.T) {
	t.Parallel()

	cfg := config.D2Config{
		Pad:    64,
		Font:   "SourceSansPro",
		Layout: "elk",
	}
	target, err := NewTarget(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	schema := domain.Schema{
		Services: []domain.Service{
			{
				Info: domain.ServiceInfo{
					Name: "Test Service",
				},
			},
		},
	}
	asyncEdges := []domain.AsyncEdge{}

	diagram, err := target.GenerateOverviewDiagram(ctx, schema, asyncEdges, "Test System")
	require.NoError(t, err)
	assert.NotEmpty(t, diagram)
	assert.Contains(t, string(diagram), "<svg")
}

func TestTarget_GenerateServiceRelationshipsDiagram(t *testing.T) {
	t.Parallel()

	cfg := config.D2Config{
		Pad:    64,
		Font:   "SourceSansPro",
		Layout: "elk",
	}
	target, err := NewTarget(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	service := domain.Service{
		Info: domain.ServiceInfo{
			Name: "Test Service",
		},
		Relationships: []domain.Relationship{
			{
				Action:      domain.RelationshipActionUses,
				Participant: "Database",
				Technology:  "PostgreSQL",
			},
		},
	}
	allServices := []domain.Service{service}
	asyncEdges := []domain.AsyncEdge{}

	diagram, err := target.GenerateServiceRelationshipsDiagram(ctx, service, allServices, asyncEdges)
	require.NoError(t, err)
	assert.NotEmpty(t, diagram)
	assert.Contains(t, string(diagram), "<svg")
}

func TestTarget_GenerateSystemDiagram(t *testing.T) {
	t.Parallel()

	cfg := config.D2Config{
		Pad:    64,
		Font:   "SourceSansPro",
		Layout: "elk",
	}
	target, err := NewTarget(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	schema := domain.Schema{
		Services: []domain.Service{
			{
				Info: domain.ServiceInfo{
					Name:   "Test Service",
					System: "Test System",
				},
			},
		},
	}
	asyncEdges := []domain.AsyncEdge{}

	diagram, err := target.GenerateSystemDiagram(ctx, schema, "Test System", asyncEdges)
	require.NoError(t, err)
	assert.NotEmpty(t, diagram)
	assert.Contains(t, string(diagram), "<svg")
}
