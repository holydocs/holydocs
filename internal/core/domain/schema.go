// Package domain provides domain entities and interfaces for the holydocs application.
package domain

import (
	"context"
	"time"

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
