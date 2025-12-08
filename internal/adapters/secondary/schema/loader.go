// Package schema provides functionality for extracting and formatting service flow schemas.
package schema

import (
	"context"
	"errors"
	"fmt"

	"github.com/holydocs/holydocs/internal/core/domain"
	"github.com/holydocs/messageflow/pkg/messageflow"
	mfschema "github.com/holydocs/messageflow/pkg/schema"
	"github.com/holydocs/servicefile/pkg/servicefile"
	do "github.com/samber/do/v2"
)

// Errors.
var (
	ErrServiceFileLoadFailed = errors.New("failed to load service file")
	ErrAsyncAPILoadFailed    = errors.New("failed to load AsyncAPI files")
)

type Loader struct{}

func NewLoader(_ do.Injector) (*Loader, error) {
	return &Loader{}, nil
}

// Load loads schemas from ServiceFile and AsyncAPI files and merges them.
func (l *Loader) Load(ctx context.Context, serviceFilesPaths, asyncapiFilesPaths []string) (domain.Schema, error) {
	var schemas []domain.Schema

	servicefileSchemas, err := l.loadServiceFiles(serviceFilesPaths)
	if err != nil {
		return domain.Schema{}, fmt.Errorf("loading service files: %w", err)
	}
	schemas = append(schemas, servicefileSchemas...)

	if len(asyncapiFilesPaths) > 0 {
		asyncapiSchemas, err := l.loadAsyncAPIFiles(ctx, asyncapiFilesPaths)
		if err != nil {
			return domain.Schema{}, fmt.Errorf("loading AsyncAPI files: %w", err)
		}
		schemas = append(schemas, asyncapiSchemas)
	}

	if len(schemas) == 0 {
		return domain.Schema{}, nil
	}

	return domain.MergeSchemas(schemas...), nil
}

func (l *Loader) loadServiceFiles(serviceFilesPaths []string) ([]domain.Schema, error) {
	schemas := make([]domain.Schema, 0, len(serviceFilesPaths))

	for _, path := range serviceFilesPaths {
		sf, err := servicefile.Load(path)
		if err != nil {
			return nil, fmt.Errorf("%w %s: %w", ErrServiceFileLoadFailed, path, err)
		}

		schemas = append(schemas, l.convertServiceFileToHolydocs(sf))
	}

	return schemas, nil
}

func (l *Loader) convertServiceFileToHolydocs(sf *servicefile.ServiceFile) domain.Schema {
	relationships := make([]domain.Relationship, 0, len(sf.Relationships))

	for _, rel := range sf.Relationships {
		relationships = append(relationships, domain.Relationship{
			Action:      domain.RelationshipAction(rel.Action),
			Participant: rel.Participant,
			Description: rel.Description,
			Technology:  rel.Technology,
			Proto:       rel.Proto,
			Tags:        append([]string(nil), rel.Tags...),
			External:    rel.External,
			Person:      rel.Person,
		})
	}

	service := domain.Service{
		Info: domain.ServiceInfo{
			Name:        sf.Info.Name,
			Description: sf.Info.Description,
			System:      sf.Info.System,
			Owner:       sf.Info.Owner,
			Repository:  sf.Info.Repository,
			Tags:        append([]string(nil), sf.Info.Tags...),
		},
		Relationships: relationships,
	}

	return domain.Schema{
		Services: []domain.Service{service},
	}
}

func (l *Loader) loadAsyncAPIFiles(ctx context.Context, asyncapiFilesPaths []string) (domain.Schema, error) {
	mfSchema, err := mfschema.Load(ctx, asyncapiFilesPaths)
	if err != nil {
		return domain.Schema{}, fmt.Errorf("%w: %w", ErrAsyncAPILoadFailed, err)
	}

	return l.convertMessageFlowToHolydocs(mfSchema), nil
}

func (l *Loader) convertMessageFlowToHolydocs(mfSchema messageflow.Schema) domain.Schema {
	holydocsServices := make([]domain.Service, 0, len(mfSchema.Services))

	for _, mfService := range mfSchema.Services {
		operations := l.convertMessageFlowOperations(mfService.Operation)
		service := domain.Service{
			Info: domain.ServiceInfo{
				Name:        mfService.Name,
				Description: mfService.Description,
			},
			Operation: operations,
		}
		holydocsServices = append(holydocsServices, service)
	}

	return domain.Schema{
		Services: holydocsServices,
	}
}

func (l *Loader) convertMessageFlowOperations(mfOperations []messageflow.Operation) []domain.Operation {
	operations := make([]domain.Operation, 0, len(mfOperations))
	for _, op := range mfOperations {
		operation := domain.Operation{
			Action: domain.OperationAction(op.Action),
			Channel: domain.Channel{
				Name: op.Channel.Name,
				Message: domain.Message{
					Name:    op.Channel.Message.Name,
					Payload: op.Channel.Message.Payload,
				},
			},
		}
		if op.Reply != nil {
			operation.Reply = &domain.Channel{
				Name: op.Reply.Name,
				Message: domain.Message{
					Name:    op.Reply.Message.Name,
					Payload: op.Reply.Message.Payload,
				},
			}
		}
		operations = append(operations, operation)
	}

	return operations
}
