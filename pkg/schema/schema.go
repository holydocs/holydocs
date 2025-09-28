// Package schema provides functionality for extracting and formatting service flow schemas.
package schema

import (
	"context"
	"errors"
	"fmt"

	"github.com/holydocs/holydocs/pkg/holydocs"
	"github.com/holydocs/messageflow/pkg/messageflow"
	mfschema "github.com/holydocs/messageflow/pkg/schema"
	"github.com/holydocs/servicefile/pkg/servicefile"
)

// Errors.
var (
	ErrServiceFileLoadFailed  = errors.New("failed to load service file")
	ErrAsyncAPILoadFailed     = errors.New("failed to load AsyncAPI files")
	ErrSchemaConversionFailed = errors.New("failed to convert schema")
)

func Load(ctx context.Context, serviceFilesPaths, asyncapiFilesPaths []string) (holydocs.Schema, error) {
	var schemas []holydocs.Schema

	servicefileSchemas, err := loadServiceFiles(serviceFilesPaths)
	if err != nil {
		return holydocs.Schema{}, fmt.Errorf("loading service files: %w", err)
	}
	schemas = append(schemas, servicefileSchemas...)

	if len(asyncapiFilesPaths) > 0 {
		asyncapiSchemas, err := loadAsyncAPIFiles(ctx, asyncapiFilesPaths)
		if err != nil {
			return holydocs.Schema{}, fmt.Errorf("loading AsyncAPI files: %w", err)
		}
		schemas = append(schemas, asyncapiSchemas)
	}

	if len(schemas) == 0 {
		return holydocs.Schema{}, nil
	}

	return holydocs.MergeSchemas(schemas...), nil
}

func loadServiceFiles(serviceFilesPaths []string) ([]holydocs.Schema, error) {
	schemas := make([]holydocs.Schema, 0, len(serviceFilesPaths))

	for _, path := range serviceFilesPaths {
		sf, err := servicefile.Load(path)
		if err != nil {
			return nil, fmt.Errorf("%w %s: %w", ErrServiceFileLoadFailed, path, err)
		}

		schemas = append(schemas, convertServiceFileToHolydocs(sf))
	}

	return schemas, nil
}

func convertServiceFileToHolydocs(sf *servicefile.ServiceFile) holydocs.Schema {
	relationships := make([]holydocs.Relationship, 0, len(sf.Relationships))

	for _, rel := range sf.Relationships {
		relationships = append(relationships, holydocs.Relationship{
			Action:      holydocs.RelationshipAction(rel.Action),
			Participant: rel.Participant,
			Description: rel.Description,
			Technology:  rel.Technology,
			Proto:       rel.Proto,
			Tags:        append([]string(nil), rel.Tags...),
			External:    rel.External,
			Person:      rel.Person,
		})
	}

	service := holydocs.Service{
		Info: holydocs.ServiceInfo{
			Name:        sf.Info.Name,
			Description: sf.Info.Description,
			System:      sf.Info.System,
			Owner:       sf.Info.Owner,
			Repository:  sf.Info.Repository,
			Tags:        append([]string(nil), sf.Info.Tags...),
		},
		Relationships: relationships,
	}

	return holydocs.Schema{
		Services: []holydocs.Service{service},
	}
}

func loadAsyncAPIFiles(ctx context.Context, asyncapiFilesPaths []string) (holydocs.Schema, error) {
	mfSchema, err := mfschema.Load(ctx, asyncapiFilesPaths)
	if err != nil {
		return holydocs.Schema{}, fmt.Errorf("%w: %w", ErrAsyncAPILoadFailed, err)
	}

	return convertMessageFlowToHolydocs(mfSchema), nil
}

func convertMessageFlowToHolydocs(mfSchema messageflow.Schema) holydocs.Schema {
	holydocsServices := make([]holydocs.Service, 0, len(mfSchema.Services))

	for _, mfService := range mfSchema.Services {
		operations := convertMessageFlowOperations(mfService.Operation)
		service := holydocs.Service{
			Info: holydocs.ServiceInfo{
				Name:        mfService.Name,
				Description: mfService.Description,
			},
			Operation: operations,
		}
		holydocsServices = append(holydocsServices, service)
	}

	return holydocs.Schema{
		Services: holydocsServices,
	}
}

func convertMessageFlowOperations(mfOperations []messageflow.Operation) []holydocs.Operation {
	operations := make([]holydocs.Operation, 0, len(mfOperations))
	for _, op := range mfOperations {
		operation := holydocs.Operation{
			Action: holydocs.OperationAction(op.Action),
			Channel: holydocs.Channel{
				Name: op.Channel.Name,
				Message: holydocs.Message{
					Name:    op.Channel.Message.Name,
					Payload: op.Channel.Message.Payload,
				},
			},
		}
		if op.Reply != nil {
			operation.Reply = &holydocs.Channel{
				Name: op.Reply.Name,
				Message: holydocs.Message{
					Name:    op.Reply.Message.Name,
					Payload: op.Reply.Message.Payload,
				},
			}
		}
		operations = append(operations, operation)
	}

	return operations
}
