package schema

import (
	"context"
	"testing"

	"github.com/holydocs/holydocs/internal/core/app"
	"github.com/holydocs/holydocs/internal/core/domain"
	do "github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestInjector() do.Injector {
	injector := do.New()
	do.Provide(injector, func(i do.Injector) (*app.App, error) {
		return app.NewApp(), nil
	})

	return injector
}

func TestLoad(t *testing.T) {
	tests := getLoadTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runLoadTestCase(t, tt)
		})
	}
}

type loadTestCase struct {
	name                string
	serviceFilesPaths   []string
	asyncapiFilesPaths  []string
	expectedServices    int
	expectedError       bool
	expectedErrorString string
}

func getLoadTestCases() []loadTestCase {
	return []loadTestCase{
		{
			name:               "empty paths",
			serviceFilesPaths:  []string{},
			asyncapiFilesPaths: []string{},
			expectedServices:   0,
			expectedError:      false,
		},
		{
			name:               "servicefile only",
			serviceFilesPaths:  []string{"testdata/analytics.servicefile.yml"},
			asyncapiFilesPaths: []string{},
			expectedServices:   1,
			expectedError:      false,
		},
		{
			name:               "asyncapi only",
			serviceFilesPaths:  []string{},
			asyncapiFilesPaths: []string{"testdata/user.asyncapi.yaml"},
			expectedServices:   1,
			expectedError:      false,
		},
		{
			name:               "both servicefile and asyncapi",
			serviceFilesPaths:  []string{"testdata/analytics.servicefile.yml"},
			asyncapiFilesPaths: []string{"testdata/user.asyncapi.yaml"},
			expectedServices:   2,
			expectedError:      false,
		},
		{
			name:                "non-existent servicefile",
			serviceFilesPaths:   []string{"testdata/nonexistent.yml"},
			asyncapiFilesPaths:  []string{},
			expectedServices:    0,
			expectedError:       true,
			expectedErrorString: "loading service files",
		},
		{
			name:                "non-existent asyncapi",
			serviceFilesPaths:   []string{},
			asyncapiFilesPaths:  []string{"testdata/nonexistent.yml"},
			expectedServices:    0,
			expectedError:       true,
			expectedErrorString: "loading AsyncAPI files",
		},
	}
}

func runLoadTestCase(t *testing.T, tt loadTestCase) {
	ctx := context.Background()
	injector := do.New()
	do.Provide(injector, func(i do.Injector) (*app.App, error) {
		return app.NewApp(), nil
	})
	loader, err := NewLoader(injector)
	require.NoError(t, err)
	schema, err := loader.Load(ctx, tt.serviceFilesPaths, tt.asyncapiFilesPaths)

	if tt.expectedError {
		require.Error(t, err)
		if tt.expectedErrorString != "" {
			assert.Contains(t, err.Error(), tt.expectedErrorString)
		}

		return
	}

	require.NoError(t, err)
	assert.Len(t, schema.Services, tt.expectedServices)
}

func TestLoad_ServiceFileContent(t *testing.T) {
	ctx := context.Background()
	injector := do.New()
	do.Provide(injector, func(i do.Injector) (*app.App, error) {
		return app.NewApp(), nil
	})
	loader, err := NewLoader(injector)
	require.NoError(t, err)
	schema, err := loader.Load(ctx, []string{"testdata/analytics.servicefile.yml"}, []string{})
	require.NoError(t, err)
	require.Len(t, schema.Services, 1)

	service := schema.Services[0]
	assert.Equal(t, "Analytics Service", service.Info.Name)
	assert.Contains(t, service.Info.Description, "analytics events")
	assert.Len(t, service.Relationships, 2)

	// Check the first relationship (uses clickhouse)
	rel1 := service.Relationships[0]
	assert.Equal(t, domain.RelationshipActionReplies, rel1.Action)
	assert.Equal(t, "Data Analyst", rel1.Participant)
	assert.Equal(t, "http-server", rel1.Technology)
	assert.True(t, rel1.Person)

	// Check the second relationship (replies Data Analyst)
	rel2 := service.Relationships[1]
	assert.Equal(t, domain.RelationshipActionUses, rel2.Action)
	assert.Equal(t, "clickhouse", rel2.Participant)
	assert.Equal(t, "ClickHouse", rel2.Technology)
}

func TestLoad_AsyncAPIContent(t *testing.T) {
	ctx := context.Background()
	injector := setupTestInjector()
	loader, err := NewLoader(injector)
	require.NoError(t, err)
	schema, err := loader.Load(ctx, []string{}, []string{"testdata/user.asyncapi.yaml"})
	require.NoError(t, err)
	require.Len(t, schema.Services, 1)

	service := schema.Services[0]
	assert.Equal(t, "User Service", service.Info.Name)
	assert.Contains(t, service.Info.Description, "A service that manages user information")
	assert.Len(t, service.Operation, 4) // 4 operations from the user.asyncapi.yaml

	// Check that we have operations
	assert.NotEmpty(t, service.Operation)

	// Verify we have both send and receive operations
	var hasSend, hasReceive bool
	for _, op := range service.Operation {
		if op.Action == domain.ActionSend {
			hasSend = true
		}
		if op.Action == domain.ActionReceive {
			hasReceive = true
		}
	}
	assert.True(t, hasSend, "Should have send operations")
	assert.True(t, hasReceive, "Should have receive operations")
}

func TestLoad_MultipleFiles(t *testing.T) {
	ctx := context.Background()
	injector := setupTestInjector()
	loader, err := NewLoader(injector)
	require.NoError(t, err)
	schema, err := loader.Load(ctx,
		[]string{"testdata/analytics.servicefile.yml"},
		[]string{"testdata/user.asyncapi.yaml"})
	require.NoError(t, err)
	require.Len(t, schema.Services, 2)

	// Find servicefile service
	var servicefileService *domain.Service
	var asyncapiService *domain.Service

	for i := range schema.Services {
		switch schema.Services[i].Info.Name {
		case "Analytics Service":
			servicefileService = &schema.Services[i]
		case "User Service":
			asyncapiService = &schema.Services[i]
		}
	}

	require.NotNil(t, servicefileService, "ServiceFile service not found")
	require.NotNil(t, asyncapiService, "AsyncAPI service not found")

	// Verify ServiceFile service has relationships
	assert.Len(t, servicefileService.Relationships, 2)
	assert.Empty(t, servicefileService.Operation)

	// Verify AsyncAPI service has operations
	assert.Empty(t, asyncapiService.Relationships)
	assert.Len(t, asyncapiService.Operation, 4)
}

func TestLoad_InvalidServiceFile(t *testing.T) {
	ctx := context.Background()
	injector := setupTestInjector()
	loader, err := NewLoader(injector)
	require.NoError(t, err)
	_, err = loader.Load(ctx, []string{"testdata/invalid-servicefile.yml"}, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading service files")
}

func TestLoad_InvalidAsyncAPI(t *testing.T) {
	ctx := context.Background()
	injector := setupTestInjector()
	loader, err := NewLoader(injector)
	require.NoError(t, err)
	_, err = loader.Load(ctx, []string{}, []string{"testdata/nonexistent.asyncapi.yaml"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading AsyncAPI files")
}
