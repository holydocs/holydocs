// Package app provides the core application logic for holydocs.
package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
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
		holydocsTarget domain.Target,
		messageflowSchema messageflow.Schema,
		messageflowTarget messageflow.Target,
		cfg *config.Config,
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

// SortSchema sorts the services, their relationships, and operations in a consistent order.
func (a *App) SortSchema(s *domain.Schema) {
	for i := range s.Services {
		sort.Slice(s.Services[i].Relationships, func(j, k int) bool {
			rel1 := s.Services[i].Relationships[j]
			rel2 := s.Services[i].Relationships[k]

			if rel1.Action != rel2.Action {
				return rel1.Action < rel2.Action
			}

			if rel1.Participant != rel2.Participant {
				return rel1.Participant < rel2.Participant
			}

			return rel1.Technology < rel2.Technology
		})

		sort.Slice(s.Services[i].Operation, func(j, k int) bool {
			op1 := s.Services[i].Operation[j]
			op2 := s.Services[i].Operation[k]

			if op1.Action != op2.Action {
				return op1.Action < op2.Action
			}

			return op1.Channel.Name < op2.Channel.Name
		})
	}

	sort.Slice(s.Services, func(i, j int) bool {
		return s.Services[i].Info.Name < s.Services[j].Info.Name
	})
}

// MergeSchemas combines multiple Schema objects into a single Schema.
func (a *App) MergeSchemas(schemas ...domain.Schema) domain.Schema {
	if len(schemas) == 0 {
		return domain.Schema{Services: []domain.Service{}}
	}

	serviceMap := make(map[string]domain.Service)

	for _, schema := range schemas {
		for _, service := range schema.Services {
			name := strings.TrimSpace(service.Info.Name)
			if name == "" {
				continue
			}

			if existingService, exists := serviceMap[name]; exists {
				serviceMap[name] = a.mergeServices(existingService, service)

				continue
			}

			serviceMap[name] = service
		}
	}

	mergedServices := make([]domain.Service, 0, len(serviceMap))
	for _, service := range serviceMap {
		mergedServices = append(mergedServices, a.normalizeService(service))
	}

	result := domain.Schema{Services: mergedServices}
	a.SortSchema(&result)

	return result
}

func (a *App) normalizeService(s domain.Service) domain.Service {
	if len(s.Info.Tags) > 0 {
		s.Info.Tags = a.uniqueStrings(s.Info.Tags)
	}

	for i := range s.Relationships {
		if len(s.Relationships[i].Tags) > 0 {
			s.Relationships[i].Tags = a.uniqueStrings(s.Relationships[i].Tags)
		}
	}

	return s
}

func (a *App) mergeServices(base, incoming domain.Service) domain.Service {
	merged := base

	merged.Info = a.mergeServiceInfo(base.Info, incoming.Info)
	merged.Relationships = a.mergeRelationships(base.Relationships, incoming.Relationships)
	merged.Operation = a.mergeOperations(base.Operation, incoming.Operation)

	return merged
}

func (a *App) mergeServiceInfo(base, incoming domain.ServiceInfo) domain.ServiceInfo {
	merged := base

	if merged.Name == "" {
		merged.Name = incoming.Name
	}

	merged.Description = a.chooseMoreInformative(incoming.Description, merged.Description)

	if merged.System == "" {
		merged.System = incoming.System
	}

	if merged.Owner == "" {
		merged.Owner = incoming.Owner
	}

	if merged.Repository == "" {
		merged.Repository = incoming.Repository
	}

	if len(incoming.Tags) > 0 {
		merged.Tags = append(merged.Tags, incoming.Tags...)
	}

	return merged
}

func (a *App) mergeRelationships(existing, incoming []domain.Relationship) []domain.Relationship {
	if len(incoming) == 0 {
		return existing
	}

	relMap := make(map[string]domain.Relationship, len(existing)+len(incoming))

	for _, rel := range existing {
		key := a.relationshipSignature(rel)
		relMap[key] = rel
	}

	for _, rel := range incoming {
		key := a.relationshipSignature(rel)
		if current, ok := relMap[key]; ok {
			updated := current
			updated.Description = a.chooseMoreInformative(rel.Description, current.Description)
			if rel.Technology != "" {
				updated.Technology = rel.Technology
			}
			if rel.Proto != "" {
				updated.Proto = rel.Proto
			}
			if rel.External {
				updated.External = true
			}
			if len(rel.Tags) > 0 {
				updated.Tags = append(updated.Tags, rel.Tags...)
			}
			relMap[key] = updated

			continue
		}

		relMap[key] = rel
	}

	merged := make([]domain.Relationship, 0, len(relMap))
	for _, rel := range relMap {
		merged = append(merged, rel)
	}

	return merged
}

func (a *App) mergeOperations(existing, incoming []domain.Operation) []domain.Operation {
	if len(incoming) == 0 {
		return existing
	}

	opMap := make(map[string]domain.Operation, len(existing)+len(incoming))

	for _, op := range existing {
		key := a.operationSignature(op)
		opMap[key] = op
	}

	for _, op := range incoming {
		key := a.operationSignature(op)
		if current, ok := opMap[key]; ok {
			updated := current
			if updated.Reply == nil && op.Reply != nil {
				reply := *op.Reply
				updated.Reply = &reply
			}
			opMap[key] = updated

			continue
		}

		opMap[key] = op
	}

	merged := make([]domain.Operation, 0, len(opMap))
	for _, op := range opMap {
		merged = append(merged, op)
	}

	return merged
}

func (a *App) relationshipSignature(rel domain.Relationship) string {
	return fmt.Sprintf("%s|%s|%s|%s", rel.Action, rel.Participant, rel.Technology, rel.Proto)
}

func (a *App) operationSignature(op domain.Operation) string {
	replyName := ""
	if op.Reply != nil {
		replyName = op.Reply.Name
	}

	return fmt.Sprintf(
		"%s|%s|%s|%s|%s",
		op.Action,
		op.Channel.Name,
		op.Channel.Message.Name,
		op.Channel.Message.Payload,
		replyName,
	)
}

func (a *App) chooseMoreInformative(candidate, current string) string {
	cc := strings.TrimSpace(candidate)
	cp := strings.TrimSpace(current)

	if cc == "" {
		return current
	}

	if cp == "" {
		return candidate
	}

	if len(cc) > len(cp) {
		return candidate
	}

	return current
}

func (a *App) uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, v := range values {
		val := strings.TrimSpace(v)
		if val == "" {
			continue
		}
		if _, exists := seen[val]; exists {
			continue
		}
		seen[val] = struct{}{}
		result = append(result, val)
	}

	return result
}

// CompareSchemas compares two schemas and returns a changelog of differences.
func (a *App) CompareSchemas(oldSchema, newSchema domain.Schema) domain.Changelog {
	changes := []domain.Change{}
	now := time.Now()

	oldServices := make(map[string]domain.Service)
	newServices := make(map[string]domain.Service)

	for _, service := range oldSchema.Services {
		oldServices[service.Info.Name] = service
	}

	for _, service := range newSchema.Services {
		newServices[service.Info.Name] = service
	}

	for name, newService := range newServices {
		if _, exists := oldServices[name]; !exists {
			changes = append(changes, domain.Change{
				Type:      domain.ChangeTypeAdded,
				Category:  "service",
				Name:      name,
				Details:   fmt.Sprintf("'%s' was added", newService.Info.Name),
				Timestamp: now,
			})
		}
	}

	for name, oldService := range oldServices {
		if _, exists := newServices[name]; !exists {
			changes = append(changes, domain.Change{
				Type:      domain.ChangeTypeRemoved,
				Category:  "service",
				Name:      name,
				Details:   fmt.Sprintf("'%s' was removed", name),
				Timestamp: now,
			})
		} else {
			serviceChanges := a.compareServiceRelationships(oldService, newServices[name], now)
			changes = append(changes, serviceChanges...)

			operationChanges := a.compareServiceOperations(oldService, newServices[name], now)
			changes = append(changes, operationChanges...)
		}
	}

	return domain.Changelog{
		Date:    now,
		Changes: changes,
	}
}

func (a *App) compareServiceRelationships(oldService, newService domain.Service, timestamp time.Time) []domain.Change {
	oldRels := a.buildRelationshipMap(oldService.Relationships)
	newRels := a.buildRelationshipMap(newService.Relationships)

	changes := []domain.Change{}
	changes = append(changes, a.findAddedRelationships(oldRels, newRels, newService.Info.Name, timestamp)...)
	changes = append(changes, a.findRemovedAndChangedRelationships(oldRels, newRels,
		oldService.Info.Name, newService.Info.Name, timestamp)...)

	return changes
}

func (a *App) buildRelationshipMap(relationships []domain.Relationship) map[string]domain.Relationship {
	relMap := make(map[string]domain.Relationship)
	for _, rel := range relationships {
		key := a.relationshipKey(rel)
		relMap[key] = rel
	}

	return relMap
}

func (a *App) findAddedRelationships(oldRels, newRels map[string]domain.Relationship, serviceName string,
	timestamp time.Time) []domain.Change {
	changes := []domain.Change{}
	for key, newRel := range newRels {
		if _, exists := oldRels[key]; !exists {
			changes = append(changes, domain.Change{
				Type:     domain.ChangeTypeAdded,
				Category: "relationship",
				Name:     fmt.Sprintf("%s:%s", serviceName, key),
				Details: fmt.Sprintf(
					"'%s' relationship to '%s' using '%s' was added to service '%s'",
					newRel.Action, newRel.Participant, newRel.Technology, serviceName,
				),
				Timestamp: timestamp,
			})
		}
	}

	return changes
}

func (a *App) findRemovedAndChangedRelationships(oldRels, newRels map[string]domain.Relationship,
	oldServiceName, newServiceName string, timestamp time.Time) []domain.Change {
	changes := []domain.Change{}
	for key, oldRel := range oldRels {
		if _, exists := newRels[key]; !exists {
			changes = append(changes, domain.Change{
				Type:     domain.ChangeTypeRemoved,
				Category: "relationship",
				Name:     fmt.Sprintf("%s:%s", oldServiceName, key),
				Details: fmt.Sprintf(
					"'%s' relationship to '%s' using '%s' was removed from service '%s'",
					oldRel.Action, oldRel.Participant, oldRel.Technology, oldServiceName,
				),
				Timestamp: timestamp,
			})
		} else {
			changes = append(changes, a.findChangedRelationship(oldRel, newRels[key], newServiceName, key, timestamp)...)
		}
	}

	return changes
}

func (a *App) findChangedRelationship(
	oldRel, newRel domain.Relationship,
	serviceName, key string,
	timestamp time.Time,
) []domain.Change {
	if cmp.Equal(oldRel.Description, newRel.Description) {
		return nil
	}

	diff := cmp.Diff(oldRel.Description, newRel.Description)

	return []domain.Change{{
		Type:     domain.ChangeTypeChanged,
		Category: "relationship",
		Name:     fmt.Sprintf("%s:%s", serviceName, key),
		Details: fmt.Sprintf(
			"Relationship description changed for '%s' to '%s' using '%s' in service '%s'",
			newRel.Action, newRel.Participant, newRel.Technology, serviceName,
		),
		Diff:      diff,
		Timestamp: timestamp,
	}}
}

func (a *App) relationshipKey(rel domain.Relationship) string {
	return fmt.Sprintf("%s|%s|%s|%s", rel.Action, rel.Participant, rel.Technology, rel.Proto)
}

func (a *App) compareServiceOperations(oldService, newService domain.Service, timestamp time.Time) []domain.Change {
	oldOps := a.buildOperationMap(oldService.Operation)
	newOps := a.buildOperationMap(newService.Operation)

	changes := []domain.Change{}
	changes = append(changes, a.findAddedOperations(oldOps, newOps, newService.Info.Name, timestamp)...)
	changes = append(changes, a.findRemovedAndChangedOperations(oldOps, newOps,
		oldService.Info.Name, newService.Info.Name, timestamp)...)

	return changes
}

func (a *App) buildOperationMap(operations []domain.Operation) map[string]domain.Operation {
	opMap := make(map[string]domain.Operation)
	for _, op := range operations {
		key := a.operationKey(op)
		opMap[key] = op
	}

	return opMap
}

func (a *App) findAddedOperations(
	oldOps, newOps map[string]domain.Operation,
	serviceName string,
	timestamp time.Time,
) []domain.Change {
	changes := []domain.Change{}
	for key, newOp := range newOps {
		if _, exists := oldOps[key]; !exists {
			changes = append(changes, domain.Change{
				Type:     domain.ChangeTypeAdded,
				Category: "operation",
				Name:     fmt.Sprintf("%s:%s", serviceName, key),
				Details: fmt.Sprintf("'%s' on channel '%s' was added to service '%s'",
					newOp.Action, newOp.Channel.Name, serviceName),
				Timestamp: timestamp,
			})
		}
	}

	return changes
}

func (a *App) findRemovedAndChangedOperations(
	oldOps, newOps map[string]domain.Operation,
	oldServiceName, newServiceName string,
	timestamp time.Time,
) []domain.Change {
	changes := []domain.Change{}
	for key, oldOp := range oldOps {
		if newOp, exists := newOps[key]; exists {
			if oldOp.Channel.Message.Payload != newOp.Channel.Message.Payload {
				diff := cmp.Diff(oldOp.Channel.Message.Payload, newOp.Channel.Message.Payload)
				changes = append(changes, domain.Change{
					Type:     domain.ChangeTypeChanged,
					Category: "message",
					Name:     fmt.Sprintf("%s:%s", newServiceName, key),
					Details: fmt.Sprintf("Message payload changed for operation '%s' on channel '%s' in service '%s'",
						newOp.Action, newOp.Channel.Name, newServiceName),
					Diff:      diff,
					Timestamp: timestamp,
				})
			}
		} else {
			changes = append(changes, domain.Change{
				Type:     domain.ChangeTypeRemoved,
				Category: "operation",
				Name:     fmt.Sprintf("%s:%s", oldServiceName, key),
				Details: fmt.Sprintf("'%s' on channel '%s' was removed from service '%s'",
					oldOp.Action, oldOp.Channel.Name, oldServiceName),
				Timestamp: timestamp,
			})
		}
	}

	return changes
}

func (a *App) operationKey(op domain.Operation) string {
	key := fmt.Sprintf("%s:%s", op.Action, op.Channel.Name)
	if op.Reply != nil {
		key += ":" + op.Reply.Name
	}

	return key
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

	changelog, err := a.docsGenerator.Generate(ctx, schema, a.target, mfSetup.Schema, mfSetup.Target, a.config)
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
