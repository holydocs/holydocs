package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareSchemas_AddedService(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
			{
				Info: ServiceInfo{
					Name: "Service B",
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes between schemas")
	assert.Len(t, changelog.Changes, 1, "Should detect one change")

	change := changelog.Changes[0]
	assert.Equal(t, ChangeTypeAdded, change.Type, "Should be an added type")
	assert.Equal(t, "service", change.Category, "Should be a service category")
	assert.Equal(t, "Service B", change.Name, "Should have correct service name")
	assert.Equal(t, "'Service B' was added", change.Details, "Should have correct details")
}

func TestCompareSchemas_RemovedService(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
			{
				Info: ServiceInfo{
					Name: "Service B",
				},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes between schemas")
	assert.Len(t, changelog.Changes, 1, "Should detect one change")

	change := changelog.Changes[0]
	assert.Equal(t, ChangeTypeRemoved, change.Type, "Should be a removed type")
	assert.Equal(t, "service", change.Category, "Should be a service category")
	assert.Equal(t, "Service B", change.Name, "Should have correct service name")
	assert.Equal(t, "'Service B' was removed", change.Details, "Should have correct details")
}

func TestCompareSchemas_AddedRelationship(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
					},
				},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
					},
					{
						Action:      RelationshipActionUses,
						Participant: "Cache",
						Technology:  "Redis",
					},
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes between schemas")
	assert.Len(t, changelog.Changes, 1, "Should detect one change")

	change := changelog.Changes[0]
	assert.Equal(t, ChangeTypeAdded, change.Type, "Should be an added type")
	assert.Equal(t, "relationship", change.Category, "Should be a relationship category")
	assert.Equal(t, "Service A:uses|Cache|Redis|", change.Name, "Should have correct relationship key")
	assert.Equal(t, "'uses' relationship to 'Cache' using 'Redis' was added to service 'Service A'",
		change.Details, "Should have correct details")
}

func TestCompareSchemas_RemovedRelationship(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
					},
					{
						Action:      RelationshipActionUses,
						Participant: "Cache",
						Technology:  "Redis",
					},
				},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
					},
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes between schemas")
	assert.Len(t, changelog.Changes, 1, "Should detect one change")

	change := changelog.Changes[0]
	assert.Equal(t, ChangeTypeRemoved, change.Type, "Should be a removed type")
	assert.Equal(t, "relationship", change.Category, "Should be a relationship category")
	assert.Equal(t, "Service A:uses|Cache|Redis|", change.Name, "Should have correct relationship key")
	assert.Equal(t, "'uses' relationship to 'Cache' using 'Redis' was removed from service 'Service A'",
		change.Details, "Should have correct details")
}

func TestCompareSchemas_ChangedRelationship(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
						Description: "Uses PostgreSQL database",
					},
				},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
						Description: "Uses PostgreSQL database for data storage",
					},
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes between schemas")
	assert.Len(t, changelog.Changes, 1, "Should detect one change")

	change := changelog.Changes[0]
	assert.Equal(t, ChangeTypeChanged, change.Type, "Should be a changed type")
	assert.Equal(t, "relationship", change.Category, "Should be a relationship category")
	assert.Equal(t, "Service A:uses|Database|PostgreSQL|", change.Name, "Should have correct relationship key")
	assert.Equal(t, "Relationship description changed for 'uses' to 'Database' using 'PostgreSQL' in service 'Service A'",
		change.Details, "Should have correct details")
}

func TestCompareSchemas_NoChanges(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
					},
				},
			},
		},
	}

	newSchema := oldSchema
	changelog := oldSchema.Compare(newSchema)

	assert.Empty(t, changelog.Changes, "Should not detect any changes for identical schemas")
}

func TestCompareSchemas_ChangelogDate(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
			{
				Info: ServiceInfo{
					Name: "Service B",
				},
			},
		},
	}

	before := time.Now()
	changelog := oldSchema.Compare(newSchema)
	after := time.Now()

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.True(t, changelog.Date.After(before) || changelog.Date.Equal(before),
		"Changelog date should be after or equal to before time")
	assert.True(t, changelog.Date.Before(after) || changelog.Date.Equal(after),
		"Changelog date should be before or equal to after time")
}

func TestRelationshipKey(t *testing.T) {
	rel := Relationship{
		Action:      RelationshipActionUses,
		Participant: "Database",
		Technology:  "PostgreSQL",
	}

	key := RelationshipKey(rel)
	expected := "uses|Database|PostgreSQL|"

	assert.Equal(t, expected, key, "Should generate correct relationship key")
}

func TestRelationshipKey_WithProto(t *testing.T) {
	rel := Relationship{
		Action:      RelationshipActionReplies,
		Participant: "User",
		Technology:  "HTTP",
		Proto:       "http",
	}

	key := RelationshipKey(rel)
	expected := "replies|User|HTTP|http"

	assert.Equal(t, expected, key, "Should generate correct relationship key with proto")
}

func TestCompareSchemas_MultipleServicesAdded(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
			{
				Info: ServiceInfo{
					Name: "Service B",
				},
			},
			{
				Info: ServiceInfo{
					Name: "Service C",
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 2, "Should detect two added services")

	// Check that both services are detected as added
	serviceNames := make(map[string]bool)
	for _, change := range changelog.Changes {
		assert.Equal(t, ChangeTypeAdded, change.Type, "Should be added type")
		assert.Equal(t, "service", change.Category, "Should be service category")
		serviceNames[change.Name] = true
	}

	assert.True(t, serviceNames["Service B"], "Should detect Service B as added")
	assert.True(t, serviceNames["Service C"], "Should detect Service C as added")
}

func TestCompareSchemas_MultipleServicesRemoved(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
			{
				Info: ServiceInfo{
					Name: "Service B",
				},
			},
			{
				Info: ServiceInfo{
					Name: "Service C",
				},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 2, "Should detect two removed services")

	// Check that both services are detected as removed
	serviceNames := make(map[string]bool)
	for _, change := range changelog.Changes {
		assert.Equal(t, ChangeTypeRemoved, change.Type, "Should be removed type")
		assert.Equal(t, "service", change.Category, "Should be service category")
		serviceNames[change.Name] = true
	}

	assert.True(t, serviceNames["Service B"], "Should detect Service B as removed")
	assert.True(t, serviceNames["Service C"], "Should detect Service C as removed")
}

func TestCompareSchemas_MultipleRelationshipsAdded(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
					},
				},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
					},
					{
						Action:      RelationshipActionUses,
						Participant: "Cache",
						Technology:  "Redis",
					},
					{
						Action:      RelationshipActionReplies,
						Participant: "User",
						Technology:  "HTTP",
						Proto:       "http",
					},
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 2, "Should detect two added relationships")

	// Check that both relationships are detected as added
	relationshipKeys := make(map[string]bool)
	for _, change := range changelog.Changes {
		assert.Equal(t, ChangeTypeAdded, change.Type, "Should be added type")
		assert.Equal(t, "relationship", change.Category, "Should be relationship category")
		relationshipKeys[change.Name] = true
	}

	assert.True(t, relationshipKeys["Service A:uses|Cache|Redis|"], "Should detect Cache relationship")
	assert.True(t, relationshipKeys["Service A:replies|User|HTTP|http"], "Should detect User relationship")
}

func TestCompareSchemas_MultipleRelationshipsRemoved(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
					},
					{
						Action:      RelationshipActionUses,
						Participant: "Cache",
						Technology:  "Redis",
					},
					{
						Action:      RelationshipActionReplies,
						Participant: "User",
						Technology:  "HTTP",
						Proto:       "http",
					},
				},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
					},
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 2, "Should detect two removed relationships")

	// Check that both relationships are detected as removed
	relationshipKeys := make(map[string]bool)
	for _, change := range changelog.Changes {
		assert.Equal(t, ChangeTypeRemoved, change.Type, "Should be removed type")
		assert.Equal(t, "relationship", change.Category, "Should be relationship category")
		relationshipKeys[change.Name] = true
	}

	assert.True(t, relationshipKeys["Service A:uses|Cache|Redis|"], "Should detect Cache relationship removal")
	assert.True(t, relationshipKeys["Service A:replies|User|HTTP|http"], "Should detect User relationship removal")
}

func TestCompareSchemas_MultipleRelationshipsChanged(t *testing.T) {
	oldSchema := createSchemaWithMultipleRelationships()
	newSchema := createSchemaWithChangedRelationships()

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 2, "Should detect two changed relationships")

	verifyChangedRelationships(t, changelog.Changes)
}

func createSchemaWithMultipleRelationships() Schema {
	return Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
						Description: "Uses PostgreSQL database",
					},
					{
						Action:      RelationshipActionUses,
						Participant: "Cache",
						Technology:  "Redis",
						Description: "Uses Redis for caching",
					},
				},
			},
		},
	}
}

func createSchemaWithChangedRelationships() Schema {
	return Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
						Description: "Uses PostgreSQL database for data storage",
					},
					{
						Action:      RelationshipActionUses,
						Participant: "Cache",
						Technology:  "Redis",
						Description: "Uses Redis for caching and session storage",
					},
				},
			},
		},
	}
}

func verifyChangedRelationships(t *testing.T, changes []Change) {
	relationshipKeys := make(map[string]bool)
	for _, change := range changes {
		assert.Equal(t, ChangeTypeChanged, change.Type, "Should be changed type")
		assert.Equal(t, "relationship", change.Category, "Should be relationship category")
		assert.NotEmpty(t, change.Diff, "Should have diff for changed relationship")
		relationshipKeys[change.Name] = true
	}

	assert.True(t, relationshipKeys["Service A:uses|Database|PostgreSQL|"], "Should detect Database relationship change")
	assert.True(t, relationshipKeys["Service A:uses|Cache|Redis|"], "Should detect Cache relationship change")
}

func TestCompareSchemas_ComplexScenario(t *testing.T) {
	oldSchema := createComplexOldSchema()
	newSchema := createComplexNewSchema()

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 5, "Should detect all changes")

	verifyComplexScenarioChanges(t, changelog.Changes)
}

func createComplexOldSchema() Schema {
	return Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
						Description: "Uses PostgreSQL database",
					},
					{
						Action:      RelationshipActionUses,
						Participant: "Cache",
						Technology:  "Redis",
					},
				},
			},
			{
				Info: ServiceInfo{
					Name: "Service B",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionReplies,
						Participant: "User",
						Technology:  "HTTP",
						Proto:       "http",
					},
				},
			},
		},
	}
}

func createComplexNewSchema() Schema {
	return Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
						Description: "Uses PostgreSQL database for data storage",
					},
					{
						Action:      RelationshipActionUses,
						Participant: "Queue",
						Technology:  "RabbitMQ",
					},
				},
			},
			{
				Info: ServiceInfo{
					Name: "Service C",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "MongoDB",
					},
				},
			},
		},
	}
}

func verifyComplexScenarioChanges(t *testing.T, changes []Change) {
	serviceChanges := 0
	relationshipChanges := 0

	for _, change := range changes {
		switch change.Category {
		case "service":
			serviceChanges++
		case "relationship":
			relationshipChanges++
		}
	}

	assert.Equal(t, 2, serviceChanges, "Should have two service changes (1 added + 1 removed)")
	assert.Equal(t, 3, relationshipChanges, "Should have three relationship changes (1 added + 1 removed + 1 changed)")
}

func TestCompareSchemas_EmptySchemas(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{},
	}

	newSchema := Schema{
		Services: []Service{},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.Empty(t, changelog.Changes, "Should not detect any changes for empty schemas")
}

func TestCompareSchemas_EmptyToNonEmpty(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 1, "Should detect one added service")

	change := changelog.Changes[0]
	assert.Equal(t, ChangeTypeAdded, change.Type, "Should be added type")
	assert.Equal(t, "service", change.Category, "Should be service category")
	assert.Equal(t, "Service A", change.Name, "Should have correct service name")
}

func TestCompareSchemas_NonEmptyToEmpty(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 1, "Should detect one removed service")

	change := changelog.Changes[0]
	assert.Equal(t, ChangeTypeRemoved, change.Type, "Should be removed type")
	assert.Equal(t, "service", change.Category, "Should be service category")
	assert.Equal(t, "Service A", change.Name, "Should have correct service name")
}

func TestCompareSchemas_ServiceWithNoRelationships(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
					},
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 1, "Should detect one added relationship")

	change := changelog.Changes[0]
	assert.Equal(t, ChangeTypeAdded, change.Type, "Should be added type")
	assert.Equal(t, "relationship", change.Category, "Should be relationship category")
	assert.Equal(t, "Service A:uses|Database|PostgreSQL|", change.Name, "Should have correct relationship key")
}

func TestCompareSchemas_ServiceWithRelationshipsToNoRelationships(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
					},
				},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 1, "Should detect one removed relationship")

	change := changelog.Changes[0]
	assert.Equal(t, ChangeTypeRemoved, change.Type, "Should be removed type")
	assert.Equal(t, "relationship", change.Category, "Should be relationship category")
	assert.Equal(t, "Service A:uses|Database|PostgreSQL|", change.Name, "Should have correct relationship key")
}

func TestCompareSchemas_RelationshipWithPersonFlag(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionReplies,
						Participant: "Data Analyst",
						Technology:  "http-server",
						Proto:       "http",
						Person:      true,
					},
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 1, "Should detect one added relationship")

	change := changelog.Changes[0]
	assert.Equal(t, ChangeTypeAdded, change.Type, "Should be added type")
	assert.Equal(t, "relationship", change.Category, "Should be relationship category")
	assert.Equal(t, "Service A:replies|Data Analyst|http-server|http", change.Name, "Should have correct relationship key")
	assert.Contains(t, change.Details, "Data Analyst", "Should mention Data Analyst in details")
}

func TestCompareSchemas_RelationshipWithEmptyProto(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
						Proto:       "",
					},
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 1, "Should detect one added relationship")

	change := changelog.Changes[0]
	assert.Equal(t, ChangeTypeAdded, change.Type, "Should be added type")
	assert.Equal(t, "relationship", change.Category, "Should be relationship category")
	assert.Equal(t, "Service A:uses|Database|PostgreSQL|", change.Name,
		"Should have correct relationship key with empty proto")
}

func TestCompareSchemas_RelationshipDescriptionWithSpecialCharacters(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
						Description: "Uses PostgreSQL database\nwith newlines and \"quotes\"",
					},
				},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
						Description: "Uses PostgreSQL database\nwith newlines and 'single quotes'",
					},
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 1, "Should detect one changed relationship")

	change := changelog.Changes[0]
	assert.Equal(t, ChangeTypeChanged, change.Type, "Should be changed type")
	assert.Equal(t, "relationship", change.Category, "Should be relationship category")
	assert.NotEmpty(t, change.Diff, "Should have diff for changed relationship")
	assert.Contains(t, change.Diff, "quotes", "Should contain the changed part in diff")
}

func TestCompareSchemas_ServiceNameWithSpecialCharacters(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service-A",
				},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service-A",
				},
			},
			{
				Info: ServiceInfo{
					Name: "Service_B",
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 1, "Should detect one added service")

	change := changelog.Changes[0]
	assert.Equal(t, ChangeTypeAdded, change.Type, "Should be added type")
	assert.Equal(t, "service", change.Category, "Should be service category")
	assert.Equal(t, "Service_B", change.Name, "Should have correct service name with underscore")
}

func TestCompareSchemas_RelationshipWithSpecialCharacters(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database-Cluster",
						Technology:  "PostgreSQL-v13",
						Proto:       "tcp/5432",
					},
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 1, "Should detect one added relationship")

	change := changelog.Changes[0]
	assert.Equal(t, ChangeTypeAdded, change.Type, "Should be added type")
	assert.Equal(t, "relationship", change.Category, "Should be relationship category")
	assert.Equal(t, "Service A:uses|Database-Cluster|PostgreSQL-v13|tcp/5432", change.Name,
		"Should handle special characters in relationship key")
}

func TestCompareSchemas_AllRelationshipActions(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
					},
					{
						Action:      RelationshipActionReplies,
						Participant: "User",
						Technology:  "HTTP",
						Proto:       "http",
					},
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 2, "Should detect two added relationships")

	// Check that both relationship actions are handled correctly
	relationshipKeys := make(map[string]bool)
	for _, change := range changelog.Changes {
		assert.Equal(t, ChangeTypeAdded, change.Type, "Should be added type")
		assert.Equal(t, "relationship", change.Category, "Should be relationship category")
		relationshipKeys[change.Name] = true
	}

	assert.True(t, relationshipKeys["Service A:uses|Database|PostgreSQL|"], "Should handle 'uses' action")
	assert.True(t, relationshipKeys["Service A:replies|User|HTTP|http"], "Should handle 'replies' action")
}

func TestCompareSchemas_ChangelogTimestampConsistency(t *testing.T) {
	oldSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
		},
	}

	newSchema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
			{
				Info: ServiceInfo{
					Name: "Service B",
				},
			},
		},
	}

	changelog := oldSchema.Compare(newSchema)

	assert.NotEmpty(t, changelog.Changes, "Should detect changes")
	assert.Len(t, changelog.Changes, 1, "Should detect one added service")

	// All changes should have the same timestamp
	expectedTimestamp := changelog.Changes[0].Timestamp
	for _, change := range changelog.Changes {
		assert.Equal(t, expectedTimestamp, change.Timestamp, "All changes should have the same timestamp")
	}

	// Changelog date should match the change timestamps
	assert.Equal(t, expectedTimestamp, changelog.Date, "Changelog date should match change timestamps")
}

func TestCompareSchemas_AsyncAPIOperations(t *testing.T) {
	t.Run("AddedOperation", testAddedOperation)
	t.Run("RemovedOperation", testRemovedOperation)
	t.Run("ChangedMessagePayload", testChangedMessagePayload)
	t.Run("MultipleOperationChanges", testMultipleOperationChanges)
}

func testAddedOperation(t *testing.T) {
	oldSchema := createBasicAnalyticsSchema()
	newSchema := createAnalyticsSchemaWithReportRequest()

	changelog := oldSchema.Compare(newSchema)

	assert.Len(t, changelog.Changes, 1, "Should detect one added operation")
	assert.Equal(t, ChangeTypeAdded, changelog.Changes[0].Type)
	assert.Equal(t, "operation", changelog.Changes[0].Category)
	assert.Contains(t, changelog.Changes[0].Details, "'receive' on channel 'analytics.report.request' was added")
}

func testRemovedOperation(t *testing.T) {
	oldSchema := createAnalyticsSchemaWithReportRequest()
	newSchema := createBasicAnalyticsSchema()

	changelog := oldSchema.Compare(newSchema)

	assert.Len(t, changelog.Changes, 1, "Should detect one removed operation")
	assert.Equal(t, ChangeTypeRemoved, changelog.Changes[0].Type)
	assert.Equal(t, "operation", changelog.Changes[0].Category)
	assert.Contains(t, changelog.Changes[0].Details, "'receive' on channel 'analytics.report.request' was removed")
}

func testChangedMessagePayload(t *testing.T) {
	oldSchema := createAnalyticsSchemaWithSeverity()
	newSchema := createAnalyticsSchemaWithConfidence()

	changelog := oldSchema.Compare(newSchema)

	assert.Len(t, changelog.Changes, 1, "Should detect one changed message")
	assert.Equal(t, ChangeTypeChanged, changelog.Changes[0].Type)
	assert.Equal(t, "message", changelog.Changes[0].Category)
	assert.Contains(t, changelog.Changes[0].Details,
		"Message payload changed for operation 'send' on channel 'analytics.insights'")
	assert.NotEmpty(t, changelog.Changes[0].Diff, "Should include diff for message change")
}

func testMultipleOperationChanges(t *testing.T) {
	oldSchema := createComplexOldAnalyticsSchema()
	newSchema := createComplexNewAnalyticsSchema()

	changelog := oldSchema.Compare(newSchema)

	assert.Len(t, changelog.Changes, 3, "Should detect multiple changes")

	// Count different types of changes
	addedCount := 0
	removedCount := 0
	changedCount := 0

	for _, change := range changelog.Changes {
		switch change.Type {
		case ChangeTypeAdded:
			addedCount++
		case ChangeTypeRemoved:
			removedCount++
		case ChangeTypeChanged:
			changedCount++
		}
	}

	assert.Equal(t, 1, addedCount, "Should have one added operation")
	assert.Equal(t, 1, removedCount, "Should have one removed operation")
	assert.Equal(t, 1, changedCount, "Should have one changed message")
}

// Helper functions to create test schemas.
func createBasicAnalyticsSchema() Schema {
	return Schema{
		Services: []Service{
			{
				Info: ServiceInfo{Name: "Analytics Service"},
				Operation: []Operation{
					{
						Action: ActionSend,
						Channel: Channel{
							Name: "analytics.insights",
							Message: Message{
								Name:    "AnalyticsInsightMessage",
								Payload: `{"insight_id": "string", "title": "string"}`,
							},
						},
					},
				},
			},
		},
	}
}

func createAnalyticsSchemaWithReportRequest() Schema {
	return Schema{
		Services: []Service{
			{
				Info: ServiceInfo{Name: "Analytics Service"},
				Operation: []Operation{
					{
						Action: ActionSend,
						Channel: Channel{
							Name: "analytics.insights",
							Message: Message{
								Name:    "AnalyticsInsightMessage",
								Payload: `{"insight_id": "string", "title": "string"}`,
							},
						},
					},
					{
						Action: ActionReceive,
						Channel: Channel{
							Name: "analytics.report.request",
							Message: Message{
								Name:    "AnalyticsReportRequestMessage",
								Payload: `{"report_id": "string", "format": "string"}`,
							},
						},
					},
				},
			},
		},
	}
}

func createAnalyticsSchemaWithSeverity() Schema {
	return Schema{
		Services: []Service{
			{
				Info: ServiceInfo{Name: "Analytics Service"},
				Operation: []Operation{
					{
						Action: ActionSend,
						Channel: Channel{
							Name: "analytics.insights",
							Message: Message{
								Name:    "AnalyticsInsightMessage",
								Payload: `{"insight_id": "string", "title": "string", "severity": "string"}`,
							},
						},
					},
				},
			},
		},
	}
}

func createAnalyticsSchemaWithConfidence() Schema {
	return Schema{
		Services: []Service{
			{
				Info: ServiceInfo{Name: "Analytics Service"},
				Operation: []Operation{
					{
						Action: ActionSend,
						Channel: Channel{
							Name: "analytics.insights",
							Message: Message{
								Name:    "AnalyticsInsightMessage",
								Payload: `{"insight_id": "string", "title": "string", "confidence": "number"}`,
							},
						},
					},
				},
			},
		},
	}
}

func createComplexOldAnalyticsSchema() Schema {
	return Schema{
		Services: []Service{
			{
				Info: ServiceInfo{Name: "Analytics Service"},
				Operation: []Operation{
					{
						Action: ActionSend,
						Channel: Channel{
							Name: "analytics.insights",
							Message: Message{
								Name:    "AnalyticsInsightMessage",
								Payload: `{"insight_id": "string", "title": "string"}`,
							},
						},
					},
					{
						Action: ActionReceive,
						Channel: Channel{
							Name: "analytics.report.request",
							Message: Message{
								Name:    "AnalyticsReportRequestMessage",
								Payload: `{"report_id": "string"}`,
							},
						},
					},
				},
			},
		},
	}
}

func createComplexNewAnalyticsSchema() Schema {
	return Schema{
		Services: []Service{
			{
				Info: ServiceInfo{Name: "Analytics Service"},
				Operation: []Operation{
					{
						Action: ActionSend,
						Channel: Channel{
							Name: "analytics.insights",
							Message: Message{
								Name:    "AnalyticsInsightMessage",
								Payload: `{"insight_id": "string", "title": "string", "confidence": "number"}`,
							},
						},
					},
					{
						Action: ActionSend,
						Channel: Channel{
							Name: "analytics.warning",
							Message: Message{
								Name:    "AnalyticsWarningMessage",
								Payload: `{"warning_id": "string", "severity": "string"}`,
							},
						},
					},
				},
			},
		},
	}
}

func TestOperationKey(t *testing.T) {
	t.Run("SimpleOperation", func(t *testing.T) {
		op := Operation{
			Action: ActionSend,
			Channel: Channel{
				Name: "analytics.insights",
				Message: Message{
					Name:    "AnalyticsInsightMessage",
					Payload: `{"insight_id": "string"}`,
				},
			},
		}

		key := OperationKey(op)
		assert.Equal(t, "send:analytics.insights", key)
	})

	t.Run("OperationWithReply", func(t *testing.T) {
		op := Operation{
			Action: ActionReceive,
			Channel: Channel{
				Name: "analytics.report.request",
				Message: Message{
					Name:    "AnalyticsReportRequestMessage",
					Payload: `{"report_id": "string"}`,
				},
			},
			Reply: &Channel{
				Name: "analytics.report.reply",
				Message: Message{
					Name:    "AnalyticsReportReplyMessage",
					Payload: `{"data": "object"}`,
				},
			},
		}

		key := OperationKey(op)
		assert.Equal(t, "receive:analytics.report.request:analytics.report.reply", key)
	})
}

func TestApp_MergeSchemas_EmptySchemas(t *testing.T) {
	t.Parallel()
	result := MergeSchemas()
	assert.Empty(t, result.Services)
}

func TestApp_MergeSchemas_SingleSchema(t *testing.T) {
	t.Parallel()
	schema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
		},
	}

	result := MergeSchemas(schema)
	assert.Len(t, result.Services, 1)
	assert.Equal(t, "Service A", result.Services[0].Info.Name)
}

func TestApp_MergeSchemas_MultipleSchemasNoOverlap(t *testing.T) {
	t.Parallel()
	schema1 := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
		},
	}
	schema2 := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service B",
				},
			},
		},
	}

	result := MergeSchemas(schema1, schema2)
	assert.Len(t, result.Services, 2)
	serviceNames := []string{result.Services[0].Info.Name, result.Services[1].Info.Name}
	assert.Contains(t, serviceNames, "Service A")
	assert.Contains(t, serviceNames, "Service B")
}

func TestApp_MergeSchemas_OverlappingServices(t *testing.T) {
	t.Parallel()
	schema1 := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name:        "Service A",
					Description: "First description",
					System:      "System 1",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
					},
				},
			},
		},
	}
	schema2 := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name:        "Service A",
					Description: "Second description (longer)",
					Owner:       "Team A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Cache",
						Technology:  "Redis",
					},
				},
			},
		},
	}

	result := MergeSchemas(schema1, schema2)
	assert.Len(t, result.Services, 1)
	service := result.Services[0]
	assert.Equal(t, "Service A", service.Info.Name)
	assert.Equal(t, "Second description (longer)", service.Info.Description)
	assert.Equal(t, "System 1", service.Info.System)
	assert.Equal(t, "Team A", service.Info.Owner)
	assert.Len(t, service.Relationships, 2)
}

func TestApp_MergeSchemas_EmptyServiceName(t *testing.T) {
	t.Parallel()
	schema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "",
				},
			},
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
		},
	}

	result := MergeSchemas(schema)
	assert.Len(t, result.Services, 1)
	assert.Equal(t, "Service A", result.Services[0].Info.Name)
}

func TestApp_MergeSchemas_WhitespaceServiceName(t *testing.T) {
	t.Parallel()
	schema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "   ",
				},
			},
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
		},
	}

	result := MergeSchemas(schema)
	assert.Len(t, result.Services, 1)
	assert.Equal(t, "Service A", result.Services[0].Info.Name)
}

func TestApp_MergeSchemas_DuplicateRelationships(t *testing.T) {
	t.Parallel()
	schema1 := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
						Description: "Original description",
					},
				},
			},
		},
	}
	schema2 := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
						Description: "Updated description (longer)",
					},
				},
			},
		},
	}

	result := MergeSchemas(schema1, schema2)
	assert.Len(t, result.Services, 1)
	assert.Len(t, result.Services[0].Relationships, 1)
	assert.Equal(t, "Updated description (longer)", result.Services[0].Relationships[0].Description)
}

func TestApp_MergeSchemas_DuplicateOperations(t *testing.T) {
	t.Parallel()
	schema1 := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Operation: []Operation{
					{
						Action: ActionSend,
						Channel: Channel{
							Name: "test.channel",
							Message: Message{
								Name:    "TestMessage",
								Payload: "{}",
							},
						},
					},
				},
			},
		},
	}
	schema2 := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Operation: []Operation{
					{
						Action: ActionSend,
						Channel: Channel{
							Name: "test.channel",
							Message: Message{
								Name:    "TestMessage",
								Payload: "{}",
							},
						},
						Reply: &Channel{
							Name: "test.reply",
							Message: Message{
								Name:    "ReplyMessage",
								Payload: "{}",
							},
						},
					},
				},
			},
		},
	}

	result := MergeSchemas(schema1, schema2)
	assert.Len(t, result.Services, 1)
	// Operations with different reply fields have different signatures, so both should be present
	assert.Len(t, result.Services[0].Operation, 2)

	opWithReply := findOperationWithReply(result.Services[0].Operation)
	require.NotNil(t, opWithReply)
	assert.Equal(t, "test.reply", opWithReply.Reply.Name)
}

func findOperationWithReply(ops []Operation) *Operation {
	for i := range ops {
		if ops[i].Reply != nil {
			return &ops[i]
		}
	}

	return nil
}

func TestApp_MergeSchemas_TagsDeduplication(t *testing.T) {
	t.Parallel()
	schema1 := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
					Tags: []string{"tag1", "tag2"},
				},
			},
		},
	}
	schema2 := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
					Tags: []string{"tag2", "tag3"},
				},
			},
		},
	}

	result := MergeSchemas(schema1, schema2)
	assert.Len(t, result.Services, 1)
	// Tags should be deduplicated
	tags := result.Services[0].Info.Tags
	assert.Contains(t, tags, "tag1")
	assert.Contains(t, tags, "tag2")
	assert.Contains(t, tags, "tag3")
}

func TestApp_SortSchema_EmptySchema(t *testing.T) {
	t.Parallel()
	schema := Schema{
		Services: []Service{},
	}

	schema.Sort()
	assert.Empty(t, schema.Services)
}

func TestApp_SortSchema_ServicesByName(t *testing.T) {
	t.Parallel()
	schema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service C",
				},
			},
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
			},
			{
				Info: ServiceInfo{
					Name: "Service B",
				},
			},
		},
	}

	schema.Sort()
	assert.Equal(t, "Service A", schema.Services[0].Info.Name)
	assert.Equal(t, "Service B", schema.Services[1].Info.Name)
	assert.Equal(t, "Service C", schema.Services[2].Info.Name)
}

func TestApp_SortSchema_Relationships(t *testing.T) {
	t.Parallel()
	schema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{
					{
						Action:      RelationshipActionUses,
						Participant: "Database",
						Technology:  "PostgreSQL",
					},
					{
						Action:      RelationshipActionReplies,
						Participant: "Client",
						Technology:  "HTTP",
					},
					{
						Action:      RelationshipActionUses,
						Participant: "Cache",
						Technology:  "Redis",
					},
				},
			},
		},
	}

	schema.Sort()
	rels := schema.Services[0].Relationships
	assert.Equal(t, RelationshipActionReplies, rels[0].Action)
	assert.Equal(t, RelationshipActionUses, rels[1].Action)
	assert.Equal(t, RelationshipActionUses, rels[2].Action)
	// Within same action, should be sorted by participant
	assert.Equal(t, "Cache", rels[1].Participant)
	assert.Equal(t, "Database", rels[2].Participant)
}

func TestApp_SortSchema_Operations(t *testing.T) {
	t.Parallel()
	schema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Operation: []Operation{
					{
						Action: ActionReceive,
						Channel: Channel{
							Name: "channel.b",
							Message: Message{
								Name:    "MessageB",
								Payload: "{}",
							},
						},
					},
					{
						Action: ActionSend,
						Channel: Channel{
							Name: "channel.a",
							Message: Message{
								Name:    "MessageA",
								Payload: "{}",
							},
						},
					},
					{
						Action: ActionReceive,
						Channel: Channel{
							Name: "channel.a",
							Message: Message{
								Name:    "MessageA",
								Payload: "{}",
							},
						},
					},
				},
			},
		},
	}

	schema.Sort()
	ops := schema.Services[0].Operation
	assert.Equal(t, ActionReceive, ops[0].Action)
	assert.Equal(t, ActionReceive, ops[1].Action)
	assert.Equal(t, ActionSend, ops[2].Action)
	// Within same action, should be sorted by channel name
	assert.Equal(t, "channel.a", ops[0].Channel.Name)
	assert.Equal(t, "channel.b", ops[1].Channel.Name)
}

func TestApp_SortSchema_ServiceWithNoRelationshipsOrOperations(t *testing.T) {
	t.Parallel()
	schema := Schema{
		Services: []Service{
			{
				Info: ServiceInfo{
					Name: "Service A",
				},
				Relationships: []Relationship{},
				Operation:     []Operation{},
			},
		},
	}

	schema.Sort()
	assert.Equal(t, "Service A", schema.Services[0].Info.Name)
	assert.Empty(t, schema.Services[0].Relationships)
	assert.Empty(t, schema.Services[0].Operation)
}
