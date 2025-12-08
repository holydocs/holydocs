// Package app provides the core application logic for holydocs.
package app

import (
	"context"
	"fmt"

	"github.com/holydocs/holydocs/internal/config"
	"github.com/holydocs/holydocs/internal/core/domain"
	"github.com/holydocs/messageflow/pkg/messageflow"
	mfschema "github.com/holydocs/messageflow/pkg/schema"
	mfd2 "github.com/holydocs/messageflow/pkg/schema/target/d2"
)

// SchemaLoader defines the interface for loading schemas from external sources.
type SchemaLoader interface {
	Load(ctx context.Context, serviceFilesPaths, asyncapiFilesPaths []string) (domain.Schema, error)
}

// TargetRenderer defines the interface for rendering formatted schemas.
type TargetRenderer interface {
	RenderSchema(ctx context.Context, fs domain.FormattedSchema) ([]byte, error)
}

// DocumentationGenerator defines the interface for generating documentation.
type DocumentationGenerator interface {
	Generate(
		ctx context.Context,
		schema domain.Schema,
		messageflowSchema messageflow.Schema,
		messageflowTarget messageflow.Target,
	) (*domain.Changelog, error)
}

// App represents the core application with all business logic.
type App struct {
	schemaLoader  SchemaLoader
	docsGenerator DocumentationGenerator
	target        domain.Target
	config        *config.Config
}

// NewApp creates a new application instance with provided dependencies.
func NewApp(
	schemaLoader SchemaLoader,
	docsGenerator DocumentationGenerator,
	target domain.Target,
	config *config.Config,
) *App {
	return &App{
		schemaLoader:  schemaLoader,
		docsGenerator: docsGenerator,
		target:        target,
		config:        config,
	}
}

// GenerateDocumentation generates documentation from the provided specification files.
func (a *App) GenerateDocumentation(
	ctx context.Context,
	req domain.GenerateDocumentationRequest,
) (domain.GenerateDocumentationReply, error) {
	schema, err := a.schemaLoader.Load(ctx, req.ServiceFilesPaths, req.AsyncAPIFilesPaths)
	if err != nil {
		return domain.GenerateDocumentationReply{}, fmt.Errorf("loading schema from files: %w", err)
	}

	mfSetup, err := createMessageFlowSetup(ctx, req.AsyncAPIFilesPaths)
	if err != nil {
		return domain.GenerateDocumentationReply{}, fmt.Errorf("setting up message flow target: %w", err)
	}

	changelog, err := a.docsGenerator.Generate(ctx, schema, mfSetup.Schema, mfSetup.Target)
	if err != nil {
		return domain.GenerateDocumentationReply{}, fmt.Errorf("generating documentation: %w", err)
	}

	return domain.GenerateDocumentationReply{
		Changelog: changelog,
	}, nil
}

func createMessageFlowSetup(
	ctx context.Context,
	asyncAPIFilesPaths []string,
) (domain.MessageFlowSetup, error) {
	if len(asyncAPIFilesPaths) == 0 {
		return domain.MessageFlowSetup{}, nil
	}

	mfSchema, err := mfschema.Load(ctx, asyncAPIFilesPaths)
	if err != nil {
		return domain.MessageFlowSetup{}, fmt.Errorf("loading messageflow schema: %w", err)
	}

	mfTarget, err := mfd2.NewTarget()
	if err != nil {
		return domain.MessageFlowSetup{}, fmt.Errorf("creating messageflow D2 target: %w", err)
	}

	return domain.MessageFlowSetup{
		Schema: mfSchema,
		Target: mfTarget,
	}, nil
}
