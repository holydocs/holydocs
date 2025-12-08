// Package domain provides domain entities and interfaces for the holydocs application.
package domain

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/holydocs/messageflow/pkg/messageflow"
)

// TargetType represents the type of target format for schema conversion.
type TargetType string

// FormatMode represents the mode of format for schema.
type FormatMode string

// Format modes.
const (
	FormatModeServiceRelationships = FormatMode("service_relationships")
	FormatModeOverview             = FormatMode("overview")
)

// FormatOptions defines options for formatting schemas.
type FormatOptions struct {
	Mode        FormatMode
	Service     string
	Technology  string
	OmitDetails bool
	AsyncEdges  []AsyncEdge
}

// Schema defines the structure of a service flow schema containing services and their relationships.
type Schema struct {
	Services []Service `json:"services"`
}

// Service represents a service in the service flow with its name and relationships.
type Service struct {
	Info          ServiceInfo    `json:"info"`
	Relationships []Relationship `json:"relationships"`
	Operation     []Operation    `json:"operations"`
}

// ServiceInfo represents info about service.
type ServiceInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	System      string   `json:"system,omitempty"`
	Owner       string   `json:"owner,omitempty"`
	Repository  string   `json:"repository,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// RelationshipAction represents the type of relationship that can exist between services.
type RelationshipAction string

// Relationship actions.
const (
	RelationshipActionUses     RelationshipAction = "uses"
	RelationshipActionRequests RelationshipAction = "requests"
	RelationshipActionReplies  RelationshipAction = "replies"
	RelationshipActionSends    RelationshipAction = "sends"
	RelationshipActionReceives RelationshipAction = "receives"
)

// Relationship represents a relationship between services with technology details.
type Relationship struct {
	Action      RelationshipAction `json:"action"`
	Participant string             `json:"participant,omitempty"`
	Description string             `json:"description,omitempty"`
	Technology  string             `json:"technology"`
	Proto       string             `json:"proto,omitempty"`
	Tags        []string           `json:"tags,omitempty"`
	External    bool               `json:"external,omitempty"`
	Person      bool               `json:"person,omitempty"`
}

// OperationAction represents the type of operation that can be performed on a channel.
type OperationAction string

// Operation actions.
const (
	ActionSend    OperationAction = "send"
	ActionReceive OperationAction = "receive"
)

// Message represents a message with a name and payload.
type Message struct {
	Name    string `json:"name"`
	Payload string `json:"payload"`
}

// Channel represents a communication channel with a name and message.
type Channel struct {
	Name    string  `json:"name"`
	Message Message `json:"message"`
}

// Operation defines an action to be performed on a channel, optionally with a reply channel.
type Operation struct {
	Action  OperationAction `json:"action"`
	Channel Channel         `json:"channel"`
	Reply   *Channel        `json:"reply,omitempty"`
}

// AsyncEdge represents an asynchronous communication edge between services.
type AsyncEdge struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Channel string `json:"channel"`
	Kind    string `json:"kind"`
}

// FormattedSchema represents a schema that has been formatted for a specific target type.
type FormattedSchema struct {
	Type TargetType `json:"type"`
	Data []byte     `json:"data"`
}

// TargetCapabilities represents the capabilities of a Target implementation.
type TargetCapabilities struct {
	Format bool
	Render bool
}

// ChangeType represents the type of change that occurred.
type ChangeType string

// Change types.
const (
	ChangeTypeAdded   ChangeType = "added"
	ChangeTypeRemoved ChangeType = "removed"
	ChangeTypeChanged ChangeType = "changed"
)

// Change represents a single change in the schema.
type Change struct {
	Type      ChangeType `json:"type"`
	Category  string     `json:"category"`
	Name      string     `json:"name"`
	Details   string     `json:"details,omitempty"`
	Diff      string     `json:"diff,omitempty"`
	Timestamp time.Time  `json:"timestamp"`
}

// Changelog represents a collection of changes with a version and date.
type Changelog struct {
	Date    time.Time `json:"date"`
	Changes []Change  `json:"changes"`
}

// Target interface defines the contract for schema formatting and rendering.
type Target interface {
	SchemaFormatter
	SchemaRenderer
	Capabilities() TargetCapabilities
}

// SchemaFormatter interface defines the contract for formatting schemas.
type SchemaFormatter interface {
	FormatSchema(ctx context.Context, s Schema, opts FormatOptions) (FormattedSchema, error)
}

// SchemaRenderer interface defines the contract for rendering formatted schemas.
type SchemaRenderer interface {
	RenderSchema(ctx context.Context, fs FormattedSchema) ([]byte, error)
}

// GenerateDocumentationRequest represents a request to generate documentation.
type GenerateDocumentationRequest struct {
	ServiceFilesPaths  []string
	AsyncAPIFilesPaths []string
	OutputDir          string
}

// GenerateDocumentationReply represents the reply from generating documentation.
type GenerateDocumentationReply struct {
	Changelog *Changelog
}

// MessageFlowSetup holds the message flow schema and target.
type MessageFlowSetup struct {
	Schema messageflow.Schema
	Target messageflow.Target
}

// Sort orders services, relationships, and operations for deterministic output.
func (s *Schema) Sort() {
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

// Merge merges the schema with additional schemas and returns a new schema.
func (s Schema) Merge(others ...Schema) Schema {
	all := append([]Schema{s}, others...)

	return mergeSchemas(all...)
}

// MergeSchemas combines multiple schemas into a single schema.
func MergeSchemas(schemas ...Schema) Schema {
	return mergeSchemas(schemas...)
}

// Compare returns a changelog describing the differences between schemas.
func (s Schema) Compare(other Schema) Changelog {
	changes := []Change{}
	now := time.Now()

	oldServices := make(map[string]Service)
	newServices := make(map[string]Service)

	for _, service := range s.Services {
		oldServices[service.Info.Name] = service
	}

	for _, service := range other.Services {
		newServices[service.Info.Name] = service
	}

	for name, newService := range newServices {
		if _, exists := oldServices[name]; !exists {
			changes = append(changes, Change{
				Type:      ChangeTypeAdded,
				Category:  "service",
				Name:      name,
				Details:   fmt.Sprintf("'%s' was added", newService.Info.Name),
				Timestamp: now,
			})
		}
	}

	for name, oldService := range oldServices {
		if _, exists := newServices[name]; !exists {
			changes = append(changes, Change{
				Type:      ChangeTypeRemoved,
				Category:  "service",
				Name:      name,
				Details:   fmt.Sprintf("'%s' was removed", name),
				Timestamp: now,
			})
		} else {
			serviceChanges := compareServiceRelationships(oldService, newServices[name], now)
			changes = append(changes, serviceChanges...)

			operationChanges := compareServiceOperations(oldService, newServices[name], now)
			changes = append(changes, operationChanges...)
		}
	}

	return Changelog{
		Date:    now,
		Changes: changes,
	}
}

func mergeSchemas(schemas ...Schema) Schema {
	if len(schemas) == 0 {
		return Schema{Services: []Service{}}
	}

	serviceMap := make(map[string]Service)

	for _, schema := range schemas {
		for _, service := range schema.Services {
			name := strings.TrimSpace(service.Info.Name)
			if name == "" {
				continue
			}

			if existingService, exists := serviceMap[name]; exists {
				serviceMap[name] = mergeServices(existingService, service)

				continue
			}

			serviceMap[name] = service
		}
	}

	mergedServices := make([]Service, 0, len(serviceMap))
	for _, service := range serviceMap {
		mergedServices = append(mergedServices, normalizeService(service))
	}

	result := Schema{Services: mergedServices}
	result.Sort()

	return result
}

func normalizeService(s Service) Service {
	if len(s.Info.Tags) > 0 {
		s.Info.Tags = uniqueStrings(s.Info.Tags)
	}

	for i := range s.Relationships {
		if len(s.Relationships[i].Tags) > 0 {
			s.Relationships[i].Tags = uniqueStrings(s.Relationships[i].Tags)
		}
	}

	return s
}

func mergeServices(base, incoming Service) Service {
	merged := base

	merged.Info = mergeServiceInfo(base.Info, incoming.Info)
	merged.Relationships = mergeRelationships(base.Relationships, incoming.Relationships)
	merged.Operation = mergeOperations(base.Operation, incoming.Operation)

	return merged
}

func mergeServiceInfo(base, incoming ServiceInfo) ServiceInfo {
	merged := base

	if merged.Name == "" {
		merged.Name = incoming.Name
	}

	merged.Description = chooseMoreInformative(incoming.Description, merged.Description)

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

func mergeRelationships(existing, incoming []Relationship) []Relationship {
	if len(incoming) == 0 {
		return existing
	}

	relMap := make(map[string]Relationship, len(existing)+len(incoming))

	for _, rel := range existing {
		key := relationshipSignature(rel)
		relMap[key] = rel
	}

	for _, rel := range incoming {
		key := relationshipSignature(rel)
		if current, ok := relMap[key]; ok {
			updated := current
			updated.Description = chooseMoreInformative(rel.Description, current.Description)
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

	merged := make([]Relationship, 0, len(relMap))
	for _, rel := range relMap {
		merged = append(merged, rel)
	}

	return merged
}

func mergeOperations(existing, incoming []Operation) []Operation {
	if len(incoming) == 0 {
		return existing
	}

	opMap := make(map[string]Operation, len(existing)+len(incoming))

	for _, op := range existing {
		key := operationSignature(op)
		opMap[key] = op
	}

	for _, op := range incoming {
		key := operationSignature(op)
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

	merged := make([]Operation, 0, len(opMap))
	for _, op := range opMap {
		merged = append(merged, op)
	}

	return merged
}

func chooseMoreInformative(candidate, current string) string {
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

func uniqueStrings(values []string) []string {
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

func relationshipSignature(rel Relationship) string {
	return fmt.Sprintf("%s|%s|%s|%s", rel.Action, rel.Participant, rel.Technology, rel.Proto)
}

func operationSignature(op Operation) string {
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

func compareServiceRelationships(oldService, newService Service, timestamp time.Time) []Change {
	oldRels := buildRelationshipMap(oldService.Relationships)
	newRels := buildRelationshipMap(newService.Relationships)

	changes := []Change{}
	changes = append(changes, findAddedRelationships(oldRels, newRels, newService.Info.Name, timestamp)...)
	changes = append(changes, findRemovedAndChangedRelationships(oldRels, newRels,
		oldService.Info.Name, newService.Info.Name, timestamp)...)

	return changes
}

func buildRelationshipMap(relationships []Relationship) map[string]Relationship {
	relMap := make(map[string]Relationship)
	for _, rel := range relationships {
		key := relationshipKey(rel)
		relMap[key] = rel
	}

	return relMap
}

func findAddedRelationships(oldRels, newRels map[string]Relationship, serviceName string,
	timestamp time.Time) []Change {
	changes := []Change{}
	for key, newRel := range newRels {
		if _, exists := oldRels[key]; !exists {
			changes = append(changes, Change{
				Type:     ChangeTypeAdded,
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

func findRemovedAndChangedRelationships(oldRels, newRels map[string]Relationship,
	oldServiceName, newServiceName string, timestamp time.Time) []Change {
	changes := []Change{}
	for key, oldRel := range oldRels {
		if _, exists := newRels[key]; !exists {
			changes = append(changes, Change{
				Type:     ChangeTypeRemoved,
				Category: "relationship",
				Name:     fmt.Sprintf("%s:%s", oldServiceName, key),
				Details: fmt.Sprintf(
					"'%s' relationship to '%s' using '%s' was removed from service '%s'",
					oldRel.Action, oldRel.Participant, oldRel.Technology, oldServiceName,
				),
				Timestamp: timestamp,
			})
		} else {
			changes = append(changes, findChangedRelationship(oldRel, newRels[key], newServiceName, key, timestamp)...)
		}
	}

	return changes
}

func findChangedRelationship(
	oldRel, newRel Relationship,
	serviceName, key string,
	timestamp time.Time,
) []Change {
	if cmp.Equal(oldRel.Description, newRel.Description) {
		return nil
	}

	diff := cmp.Diff(oldRel.Description, newRel.Description)

	return []Change{{
		Type:     ChangeTypeChanged,
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

func relationshipKey(rel Relationship) string {
	return fmt.Sprintf("%s|%s|%s|%s", rel.Action, rel.Participant, rel.Technology, rel.Proto)
}

// RelationshipKey returns the deterministic key for a relationship.
func RelationshipKey(rel Relationship) string {
	return relationshipKey(rel)
}

func compareServiceOperations(oldService, newService Service, timestamp time.Time) []Change {
	oldOps := buildOperationMap(oldService.Operation)
	newOps := buildOperationMap(newService.Operation)

	changes := []Change{}
	changes = append(changes, findAddedOperations(oldOps, newOps, newService.Info.Name, timestamp)...)
	changes = append(changes, findRemovedAndChangedOperations(oldOps, newOps,
		oldService.Info.Name, newService.Info.Name, timestamp)...)

	return changes
}

func buildOperationMap(operations []Operation) map[string]Operation {
	opMap := make(map[string]Operation)
	for _, op := range operations {
		key := operationKey(op)
		opMap[key] = op
	}

	return opMap
}

func findAddedOperations(
	oldOps, newOps map[string]Operation,
	serviceName string,
	timestamp time.Time,
) []Change {
	changes := []Change{}
	for key, newOp := range newOps {
		if _, exists := oldOps[key]; !exists {
			changes = append(changes, Change{
				Type:     ChangeTypeAdded,
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

func findRemovedAndChangedOperations(
	oldOps, newOps map[string]Operation,
	oldServiceName, newServiceName string,
	timestamp time.Time,
) []Change {
	changes := []Change{}
	for key, oldOp := range oldOps {
		if newOp, exists := newOps[key]; exists {
			if oldOp.Channel.Message.Payload != newOp.Channel.Message.Payload {
				diff := cmp.Diff(oldOp.Channel.Message.Payload, newOp.Channel.Message.Payload)
				changes = append(changes, Change{
					Type:     ChangeTypeChanged,
					Category: "message",
					Name:     fmt.Sprintf("%s:%s", newServiceName, key),
					Details: fmt.Sprintf("Message payload changed for operation '%s' on channel '%s' in service '%s'",
						newOp.Action, newOp.Channel.Name, newServiceName),
					Diff:      diff,
					Timestamp: timestamp,
				})
			}
		} else {
			changes = append(changes, Change{
				Type:     ChangeTypeRemoved,
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

func operationKey(op Operation) string {
	key := fmt.Sprintf("%s:%s", op.Action, op.Channel.Name)
	if op.Reply != nil {
		key += ":" + op.Reply.Name
	}

	return key
}

// OperationKey returns the deterministic key for an operation.
func OperationKey(op Operation) string {
	return operationKey(op)
}
