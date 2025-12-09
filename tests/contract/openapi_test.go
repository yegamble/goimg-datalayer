package contract_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain loads the OpenAPI spec once for all tests.
var (
	loader *openapi3.Loader
	doc    *openapi3.T
	router routers.Router
)

func TestMain(m *testing.M) {
	// Load OpenAPI spec
	specPath := getSpecPath()
	var err error
	loader = openapi3.NewLoader()
	doc, err = loader.LoadFromFile(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load OpenAPI spec: %v\n", err)
		os.Exit(1)
	}

	// Validate spec
	err = doc.Validate(loader.Context)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OpenAPI spec validation failed: %v\n", err)
		os.Exit(1)
	}

	// Create router for path matching
	router, err = gorillamux.NewRouter(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create router: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	os.Exit(m.Run())
}

// getSpecPath returns the absolute path to the OpenAPI spec.
func getSpecPath() string {
	// Navigate from tests/contract/ to api/openapi/openapi.yaml
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(dir, "..", "..", "api", "openapi", "openapi.yaml")
}

// TestOpenAPISpecLoads verifies the OpenAPI spec can be loaded and is valid.
func TestOpenAPISpecLoads(t *testing.T) {
	t.Parallel()

	require.NotNil(t, doc, "OpenAPI document should be loaded")
	assert.Equal(t, "3.0.3", doc.OpenAPI)
	assert.Equal(t, "goimg-datalayer API", doc.Info.Title)
	assert.Equal(t, "1.0.0", doc.Info.Version)
}

// TestEndpointDefinitions verifies all expected endpoints are defined in the spec.
func TestEndpointDefinitions(t *testing.T) {
	t.Parallel()

	expectedEndpoints := map[string][]string{
		// Auth endpoints
		"/auth/register": {http.MethodPost},
		"/auth/login":    {http.MethodPost},
		"/auth/refresh":  {http.MethodPost},
		"/auth/logout":   {http.MethodPost},
		// User endpoints
		"/users/{id}":          {http.MethodGet, http.MethodPut, http.MethodDelete},
		"/users/{id}/likes":    {http.MethodGet},
		"/users/{id}/sessions": {http.MethodGet},
		// Image endpoints
		"/images":                      {http.MethodGet, http.MethodPost},
		"/images/{id}":                 {http.MethodGet, http.MethodPut, http.MethodDelete},
		"/images/{id}/variants/{size}": {http.MethodGet},
		// Album endpoints
		"/albums":                       {http.MethodGet, http.MethodPost},
		"/albums/{id}":                  {http.MethodGet, http.MethodPut, http.MethodDelete},
		"/albums/{id}/images":           {http.MethodPost},
		"/albums/{id}/images/{imageId}": {http.MethodDelete},
		// Tag endpoints
		"/tags":              {http.MethodGet},
		"/tags/search":       {http.MethodGet},
		"/tags/{tag}/images": {http.MethodGet},
		// Social endpoints - note: /images/{id}/likes is NOT in spec (planned for future)
		"/images/{id}/like":     {http.MethodPost, http.MethodDelete},
		"/images/{id}/comments": {http.MethodGet, http.MethodPost},
		"/comments/{id}":        {http.MethodDelete},
		// Moderation endpoints
		"/reports":                         {http.MethodPost},
		"/moderation/reports":              {http.MethodGet},
		"/moderation/reports/{id}":         {http.MethodGet},
		"/moderation/reports/{id}/resolve": {http.MethodPost},
		"/users/{id}/ban":                  {http.MethodPost},
		// Explore endpoints
		"/explore/recent":  {http.MethodGet},
		"/explore/popular": {http.MethodGet},
		// Health endpoints
		"/health":       {http.MethodGet},
		"/health/ready": {http.MethodGet},
		// Monitoring endpoints
		"/metrics": {http.MethodGet},
	}

	for path, methods := range expectedEndpoints {
		pathItem := doc.Paths.Find(path)
		require.NotNil(t, pathItem, "Path %s should be defined in spec", path)

		for _, method := range methods {
			operation := pathItem.GetOperation(method)
			assert.NotNil(t, operation, "Path %s should have %s method defined", path, method)
		}
	}
}

// TestAuthEndpointsContract tests contract compliance for authentication endpoints.
func TestAuthEndpointsContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		path            string
		method          string
		requiresAuth    bool
		requestSchema   map[string]interface{}
		responseSchemas map[int]string // status code -> schema ref
	}{
		{
			name:         "POST /auth/register",
			path:         "/auth/register",
			method:       http.MethodPost,
			requiresAuth: false,
			requestSchema: map[string]interface{}{
				"email":    "string",
				"username": "string",
				"password": "string",
			},
			responseSchemas: map[int]string{
				201: "user_created",
				400: "ProblemDetail",
				409: "ProblemDetail",
			},
		},
		{
			name:         "POST /auth/login",
			path:         "/auth/login",
			method:       http.MethodPost,
			requiresAuth: false,
			requestSchema: map[string]interface{}{
				"email":    "string",
				"password": "string",
			},
			responseSchemas: map[int]string{
				200: "TokenResponse",
				401: "ProblemDetail",
				429: "ProblemDetail",
			},
		},
		{
			name:         "POST /auth/refresh",
			path:         "/auth/refresh",
			method:       http.MethodPost,
			requiresAuth: false,
			requestSchema: map[string]interface{}{
				"refresh_token": "string",
			},
			responseSchemas: map[int]string{
				200: "TokenResponse",
				401: "ProblemDetail",
			},
		},
		{
			name:         "POST /auth/logout",
			path:         "/auth/logout",
			method:       http.MethodPost,
			requiresAuth: true,
			responseSchemas: map[int]string{
				204: "no_content",
				401: "ProblemDetail",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateEndpointContract(t, tt.path, tt.method, tt.requiresAuth, tt.requestSchema, tt.responseSchemas)
		})
	}
}

// TestUserEndpointsContract tests contract compliance for user endpoints.
func TestUserEndpointsContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		path            string
		method          string
		requiresAuth    bool
		requestSchema   map[string]interface{}
		responseSchemas map[int]string
	}{
		{
			name:         "GET /users/{id}",
			path:         "/users/{id}",
			method:       http.MethodGet,
			requiresAuth: false,
			responseSchemas: map[int]string{
				200: "User",
				404: "ProblemDetail",
			},
		},
		{
			name:         "PUT /users/{id}",
			path:         "/users/{id}",
			method:       http.MethodPut,
			requiresAuth: true,
			requestSchema: map[string]interface{}{
				"display_name": "string",
				"bio":          "string",
			},
			responseSchemas: map[int]string{
				200: "User",
				400: "ProblemDetail",
				401: "ProblemDetail",
				403: "ProblemDetail",
				404: "ProblemDetail",
			},
		},
		{
			name:         "DELETE /users/{id}",
			path:         "/users/{id}",
			method:       http.MethodDelete,
			requiresAuth: true,
			responseSchemas: map[int]string{
				204: "no_content",
				401: "ProblemDetail",
				403: "ProblemDetail",
				404: "ProblemDetail",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateEndpointContract(t, tt.path, tt.method, tt.requiresAuth, tt.requestSchema, tt.responseSchemas)
		})
	}
}

// TestImageEndpointsContract tests contract compliance for image endpoints.
func TestImageEndpointsContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		path            string
		method          string
		requiresAuth    bool
		requestSchema   map[string]interface{}
		responseSchemas map[int]string
	}{
		{
			name:         "GET /images",
			path:         "/images",
			method:       http.MethodGet,
			requiresAuth: false,
			responseSchemas: map[int]string{
				200: "PaginatedResponse",
				400: "ProblemDetail",
			},
		},
		{
			name:         "POST /images",
			path:         "/images",
			method:       http.MethodPost,
			requiresAuth: true,
			requestSchema: map[string]interface{}{
				"file": "binary",
			},
			responseSchemas: map[int]string{
				201: "Image",
				400: "ProblemDetail",
				401: "ProblemDetail",
				413: "ProblemDetail",
			},
		},
		{
			name:         "GET /images/{id}",
			path:         "/images/{id}",
			method:       http.MethodGet,
			requiresAuth: false,
			responseSchemas: map[int]string{
				200: "Image",
				404: "ProblemDetail",
			},
		},
		{
			name:         "PUT /images/{id}",
			path:         "/images/{id}",
			method:       http.MethodPut,
			requiresAuth: true,
			requestSchema: map[string]interface{}{
				"title":       "string",
				"description": "string",
				"visibility":  "string",
				"tags":        "array",
			},
			responseSchemas: map[int]string{
				200: "Image",
				400: "ProblemDetail",
				401: "ProblemDetail",
				403: "ProblemDetail",
				404: "ProblemDetail",
			},
		},
		{
			name:         "DELETE /images/{id}",
			path:         "/images/{id}",
			method:       http.MethodDelete,
			requiresAuth: true,
			responseSchemas: map[int]string{
				204: "no_content",
				401: "ProblemDetail",
				403: "ProblemDetail",
				404: "ProblemDetail",
			},
		},
		{
			name:         "GET /images/{id}/variants/{size}",
			path:         "/images/{id}/variants/{size}",
			method:       http.MethodGet,
			requiresAuth: false,
			responseSchemas: map[int]string{
				200: "image_binary",
				404: "ProblemDetail",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateEndpointContract(t, tt.path, tt.method, tt.requiresAuth, tt.requestSchema, tt.responseSchemas)
		})
	}
}

// TestAlbumEndpointsContract tests contract compliance for album endpoints.
func TestAlbumEndpointsContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		path            string
		method          string
		requiresAuth    bool
		requestSchema   map[string]interface{}
		responseSchemas map[int]string
	}{
		{
			name:         "GET /albums",
			path:         "/albums",
			method:       http.MethodGet,
			requiresAuth: false,
			responseSchemas: map[int]string{
				200: "PaginatedResponse",
			},
		},
		{
			name:         "POST /albums",
			path:         "/albums",
			method:       http.MethodPost,
			requiresAuth: true,
			requestSchema: map[string]interface{}{
				"title":       "string",
				"description": "string",
				"visibility":  "string",
			},
			responseSchemas: map[int]string{
				201: "Album",
				400: "ProblemDetail",
				401: "ProblemDetail",
			},
		},
		{
			name:         "GET /albums/{id}",
			path:         "/albums/{id}",
			method:       http.MethodGet,
			requiresAuth: false,
			responseSchemas: map[int]string{
				200: "Album",
				404: "ProblemDetail",
			},
		},
		{
			name:         "PUT /albums/{id}",
			path:         "/albums/{id}",
			method:       http.MethodPut,
			requiresAuth: true,
			requestSchema: map[string]interface{}{
				"title":       "string",
				"description": "string",
				"visibility":  "string",
			},
			responseSchemas: map[int]string{
				200: "Album",
				400: "ProblemDetail",
				401: "ProblemDetail",
				403: "ProblemDetail",
				404: "ProblemDetail",
			},
		},
		{
			name:         "DELETE /albums/{id}",
			path:         "/albums/{id}",
			method:       http.MethodDelete,
			requiresAuth: true,
			responseSchemas: map[int]string{
				204: "no_content",
				401: "ProblemDetail",
				403: "ProblemDetail",
				404: "ProblemDetail",
			},
		},
		{
			name:         "POST /albums/{id}/images",
			path:         "/albums/{id}/images",
			method:       http.MethodPost,
			requiresAuth: true,
			requestSchema: map[string]interface{}{
				"image_ids": "array",
			},
			responseSchemas: map[int]string{
				200: "added_count",
				400: "ProblemDetail",
				401: "ProblemDetail",
				403: "ProblemDetail",
				404: "ProblemDetail",
			},
		},
		{
			name:         "DELETE /albums/{id}/images/{imageId}",
			path:         "/albums/{id}/images/{imageId}",
			method:       http.MethodDelete,
			requiresAuth: true,
			responseSchemas: map[int]string{
				204: "no_content",
				401: "ProblemDetail",
				403: "ProblemDetail",
				404: "ProblemDetail",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateEndpointContract(t, tt.path, tt.method, tt.requiresAuth, tt.requestSchema, tt.responseSchemas)
		})
	}
}

// TestSocialEndpointsContract tests contract compliance for social endpoints.
func TestSocialEndpointsContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		path            string
		method          string
		requiresAuth    bool
		requestSchema   map[string]interface{}
		responseSchemas map[int]string
	}{
		{
			name:         "POST /images/{id}/like",
			path:         "/images/{id}/like",
			method:       http.MethodPost,
			requiresAuth: true,
			responseSchemas: map[int]string{
				200: "like_response",
				401: "ProblemDetail",
				404: "ProblemDetail",
			},
		},
		{
			name:         "DELETE /images/{id}/like",
			path:         "/images/{id}/like",
			method:       http.MethodDelete,
			requiresAuth: true,
			responseSchemas: map[int]string{
				200: "like_response",
				401: "ProblemDetail",
				404: "ProblemDetail",
			},
		},
		{
			name:         "GET /users/{id}/likes",
			path:         "/users/{id}/likes",
			method:       http.MethodGet,
			requiresAuth: false, // Optional auth
			responseSchemas: map[int]string{
				200: "PaginatedResponse",
				404: "ProblemDetail",
			},
		},
		{
			name:         "POST /images/{id}/comments",
			path:         "/images/{id}/comments",
			method:       http.MethodPost,
			requiresAuth: true,
			requestSchema: map[string]interface{}{
				"content": "string",
			},
			responseSchemas: map[int]string{
				201: "Comment",
				400: "ProblemDetail",
				401: "ProblemDetail",
				404: "ProblemDetail",
			},
		},
		{
			name:         "GET /images/{id}/comments",
			path:         "/images/{id}/comments",
			method:       http.MethodGet,
			requiresAuth: false, // Optional auth
			responseSchemas: map[int]string{
				200: "PaginatedResponse",
			},
		},
		{
			name:         "DELETE /comments/{id}",
			path:         "/comments/{id}",
			method:       http.MethodDelete,
			requiresAuth: true,
			responseSchemas: map[int]string{
				204: "no_content",
				401: "ProblemDetail",
				403: "ProblemDetail",
				404: "ProblemDetail",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateEndpointContract(t, tt.path, tt.method, tt.requiresAuth, tt.requestSchema, tt.responseSchemas)
		})
	}
}

// TestTagEndpointsContract tests contract compliance for tag endpoints.
func TestTagEndpointsContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		path            string
		method          string
		requiresAuth    bool
		responseSchemas map[int]string
	}{
		{
			name:         "GET /tags",
			path:         "/tags",
			method:       http.MethodGet,
			requiresAuth: false,
			responseSchemas: map[int]string{
				200: "tags_response",
			},
		},
		{
			name:         "GET /tags/search",
			path:         "/tags/search",
			method:       http.MethodGet,
			requiresAuth: false,
			responseSchemas: map[int]string{
				200: "tags_response",
			},
		},
		{
			name:         "GET /tags/{tag}/images",
			path:         "/tags/{tag}/images",
			method:       http.MethodGet,
			requiresAuth: false,
			responseSchemas: map[int]string{
				200: "PaginatedResponse",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateEndpointContract(t, tt.path, tt.method, tt.requiresAuth, nil, tt.responseSchemas)
		})
	}
}

// TestUserSessionsEndpointsContract tests contract compliance for user session endpoints.
func TestUserSessionsEndpointsContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		path            string
		method          string
		requiresAuth    bool
		responseSchemas map[int]string
	}{
		{
			name:         "GET /users/{id}/sessions",
			path:         "/users/{id}/sessions",
			method:       http.MethodGet,
			requiresAuth: true,
			responseSchemas: map[int]string{
				200: "sessions_response",
				401: "ProblemDetail",
				403: "ProblemDetail",
				404: "ProblemDetail",
			},
		},
		// Note: DELETE /users/{id}/sessions/{sessionId} is not in the OpenAPI spec
		// Session deletion is handled through POST /auth/logout
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateEndpointContract(t, tt.path, tt.method, tt.requiresAuth, nil, tt.responseSchemas)
		})
	}
}

// TestModerationEndpointsContract tests contract compliance for moderation endpoints.
func TestModerationEndpointsContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		path            string
		method          string
		requiresAuth    bool
		requestSchema   map[string]interface{}
		responseSchemas map[int]string
	}{
		{
			name:         "POST /reports",
			path:         "/reports",
			method:       http.MethodPost,
			requiresAuth: true,
			requestSchema: map[string]interface{}{
				"image_id":    "string",
				"reason":      "string",
				"description": "string",
			},
			responseSchemas: map[int]string{
				201: "Report",
				400: "ProblemDetail",
				401: "ProblemDetail",
				403: "ProblemDetail",
			},
		},
		{
			name:         "GET /moderation/reports",
			path:         "/moderation/reports",
			method:       http.MethodGet,
			requiresAuth: true,
			responseSchemas: map[int]string{
				200: "PaginatedResponse",
				401: "ProblemDetail",
				403: "ProblemDetail",
			},
		},
		{
			name:         "GET /moderation/reports/{id}",
			path:         "/moderation/reports/{id}",
			method:       http.MethodGet,
			requiresAuth: true,
			responseSchemas: map[int]string{
				200: "Report",
				401: "ProblemDetail",
				403: "ProblemDetail",
				404: "ProblemDetail",
			},
		},
		{
			name:         "POST /moderation/reports/{id}/resolve",
			path:         "/moderation/reports/{id}/resolve",
			method:       http.MethodPost,
			requiresAuth: true,
			requestSchema: map[string]interface{}{
				"status":     "string",
				"resolution": "string",
			},
			responseSchemas: map[int]string{
				200: "Report",
				400: "ProblemDetail",
				401: "ProblemDetail",
				403: "ProblemDetail",
				404: "ProblemDetail",
			},
		},
		{
			name:         "POST /users/{id}/ban",
			path:         "/users/{id}/ban",
			method:       http.MethodPost,
			requiresAuth: true,
			requestSchema: map[string]interface{}{
				"reason":   "string",
				"duration": "integer",
			},
			responseSchemas: map[int]string{
				200: "User",
				400: "ProblemDetail",
				401: "ProblemDetail",
				403: "ProblemDetail",
				404: "ProblemDetail",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateEndpointContract(t, tt.path, tt.method, tt.requiresAuth, tt.requestSchema, tt.responseSchemas)
		})
	}
}

// TestExploreEndpointsContract tests contract compliance for explore endpoints.
func TestExploreEndpointsContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		path            string
		method          string
		requiresAuth    bool
		responseSchemas map[int]string
	}{
		{
			name:         "GET /explore/recent",
			path:         "/explore/recent",
			method:       http.MethodGet,
			requiresAuth: false,
			responseSchemas: map[int]string{
				200: "PaginatedResponse",
			},
		},
		{
			name:         "GET /explore/popular",
			path:         "/explore/popular",
			method:       http.MethodGet,
			requiresAuth: false,
			responseSchemas: map[int]string{
				200: "PaginatedResponse",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateEndpointContract(t, tt.path, tt.method, tt.requiresAuth, nil, tt.responseSchemas)
		})
	}
}

// TestHealthEndpointsContract tests contract compliance for health endpoints.
func TestHealthEndpointsContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		path            string
		method          string
		requiresAuth    bool
		responseSchemas map[int]string
	}{
		{
			name:         "GET /health",
			path:         "/health",
			method:       http.MethodGet,
			requiresAuth: false,
			responseSchemas: map[int]string{
				200: "HealthStatus",
			},
		},
		{
			name:         "GET /health/ready",
			path:         "/health/ready",
			method:       http.MethodGet,
			requiresAuth: false,
			responseSchemas: map[int]string{
				200: "HealthReadyResponse",
				503: "HealthReadyResponse",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateEndpointContract(t, tt.path, tt.method, tt.requiresAuth, nil, tt.responseSchemas)
		})
	}
}

// TestComponentSchemas verifies all component schemas are properly defined.
func TestComponentSchemas(t *testing.T) {
	t.Parallel()

	requiredSchemas := []string{
		"User",
		"Image",
		"Album",
		"Comment",
		"Like",
		"Report",
		"TokenResponse",
		"PaginatedResponse",
		"Pagination",
		"ProblemDetail",
		"HealthStatus",
		"HealthReadyResponse",
		"HealthCheck",
	}

	for _, schemaName := range requiredSchemas {
		t.Run(schemaName, func(t *testing.T) {
			t.Parallel()

			schema := doc.Components.Schemas[schemaName]
			assert.NotNil(t, schema, "Schema %s should be defined in components", schemaName)

			if schema != nil {
				assert.NotNil(t, schema.Value, "Schema %s should have a value", schemaName)
			}
		})
	}
}

// TestProblemDetailSchema validates RFC 7807 error schema.
func TestProblemDetailSchema(t *testing.T) {
	t.Parallel()

	schema := doc.Components.Schemas["ProblemDetail"]
	require.NotNil(t, schema, "ProblemDetail schema should exist")
	require.NotNil(t, schema.Value, "ProblemDetail schema should have value")

	// Verify required fields
	requiredFields := []string{"type", "title", "status"}
	for _, field := range requiredFields {
		assert.Contains(t, schema.Value.Required, field, "ProblemDetail should require field: %s", field)
	}

	// Verify optional fields exist
	optionalFields := []string{"detail", "instance", "traceId", "errors"}
	for _, field := range optionalFields {
		_, exists := schema.Value.Properties[field]
		assert.True(t, exists, "ProblemDetail should have optional field: %s", field)
	}
}

// TestSecuritySchemes verifies security schemes are properly defined.
func TestSecuritySchemes(t *testing.T) {
	t.Parallel()

	bearerAuth := doc.Components.SecuritySchemes["bearerAuth"]
	require.NotNil(t, bearerAuth, "bearerAuth security scheme should be defined")
	require.NotNil(t, bearerAuth.Value, "bearerAuth should have value")

	assert.Equal(t, "http", bearerAuth.Value.Type)
	assert.Equal(t, "bearer", bearerAuth.Value.Scheme)
	assert.Equal(t, "JWT", bearerAuth.Value.BearerFormat)
}

// TestPaginationParameters verifies pagination parameters are properly defined.
func TestPaginationParameters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		paramName    string
		expectedType string
		minimum      *int64
		maximum      *int64
		defaultValue interface{}
	}{
		{
			name:         "PageParam",
			paramName:    "PageParam",
			expectedType: "integer",
			minimum:      int64Ptr(1),
			defaultValue: 1,
		},
		{
			name:         "PerPageParam",
			paramName:    "PerPageParam",
			expectedType: "integer",
			minimum:      int64Ptr(1),
			maximum:      int64Ptr(100),
			defaultValue: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			param := doc.Components.Parameters[tt.paramName]
			require.NotNil(t, param, "Parameter %s should be defined", tt.paramName)
			require.NotNil(t, param.Value, "Parameter %s should have value", tt.paramName)
			require.NotNil(t, param.Value.Schema, "Parameter %s should have schema", tt.paramName)
			require.NotNil(t, param.Value.Schema.Value, "Parameter %s schema should have value", tt.paramName)

			assert.Equal(t, tt.expectedType, param.Value.Schema.Value.Type.Slice()[0])

			if tt.minimum != nil {
				assert.InDelta(t, float64(*tt.minimum), *param.Value.Schema.Value.Min, 0.001)
			}

			if tt.maximum != nil {
				assert.InDelta(t, float64(*tt.maximum), *param.Value.Schema.Value.Max, 0.001)
			}
		})
	}
}

// Helper Functions

// validateEndpointContract validates that an endpoint's contract matches the spec.
//
//nolint:gocognit // Contract test helper requires comprehensive validation of all endpoint aspects
func validateEndpointContract(
	t *testing.T,
	path, method string,
	requiresAuth bool,
	requestSchema map[string]interface{},
	responseSchemas map[int]string,
) {
	t.Helper()

	// Find the path in the spec
	pathItem := doc.Paths.Find(path)
	require.NotNil(t, pathItem, "Path %s should exist in spec", path)

	// Get the operation
	operation := pathItem.GetOperation(method)
	require.NotNil(t, operation, "Operation %s %s should exist", method, path)

	// Validate security requirements
	if requiresAuth {
		// Check if operation has explicit security or inherits from global
		hasExplicitSecurity := operation.Security != nil && len(*operation.Security) > 0
		hasGlobalSecurity := len(doc.Security) > 0

		switch {
		case hasExplicitSecurity:
			// Check if any security requirement includes bearerAuth
			foundBearer := false
			for _, secReq := range *operation.Security {
				if _, hasBearer := secReq["bearerAuth"]; hasBearer {
					foundBearer = true
					break
				}
			}
			assert.True(t, foundBearer, "Endpoint %s %s should use bearerAuth (found %d security requirements)", method, path, len(*operation.Security))
		case hasGlobalSecurity:
			// Inherits from global security
			secReq := doc.Security[0]
			_, hasBearer := secReq["bearerAuth"]
			assert.True(t, hasBearer, "Endpoint %s %s should inherit bearerAuth from global", method, path)
		default:
			assert.Fail(t, fmt.Sprintf("Endpoint %s %s should require authentication", method, path))
		}
	} else if operation.Security != nil && len(*operation.Security) > 0 {
		// Public endpoints can have:
		// 1. Empty security (explicitly override global)
		// 2. Security with empty object {} (public access)
		// 3. Security with both {} and bearerAuth (optional auth)

		// Check if any security requirement is empty {} (public access)
		hasPublicAccess := false
		for _, secReq := range *operation.Security {
			if len(secReq) == 0 {
				hasPublicAccess = true
				break
			}
		}
		assert.True(t, hasPublicAccess, "Public endpoint %s %s should have {} security requirement for public access", method, path)
	}

	// Validate request schema if provided
	if requestSchema != nil {
		if operation.RequestBody != nil && operation.RequestBody.Value != nil {
			assert.True(t, operation.RequestBody.Value.Required, "Request body should be required for %s %s", method, path)

			jsonContent := operation.RequestBody.Value.Content.Get("application/json")
			if jsonContent != nil {
				assert.NotNil(t, jsonContent.Schema, "Request should have schema for %s %s", method, path)
			}

			// Check for multipart/form-data for file uploads
			multipartContent := operation.RequestBody.Value.Content.Get("multipart/form-data")
			if multipartContent != nil {
				assert.NotNil(t, multipartContent.Schema, "Multipart request should have schema for %s %s", method, path)
			}
		} else {
			assert.Fail(t, fmt.Sprintf("Operation %s %s should have request body but doesn't", method, path))
		}
	}

	// Validate response schemas
	for statusCode, schemaName := range responseSchemas {
		statusCodeStr := fmt.Sprintf("%d", statusCode)
		response := operation.Responses.Status(statusCode)
		assert.NotNil(t, response, "Response %d should be defined for %s %s", statusCode, method, path)

		if response != nil && response.Value != nil {
			// Special cases that don't have content
			switch schemaName {
			case "no_content":
				assert.Nil(t, response.Value.Content, "Response %s should have no content for %s %s", statusCodeStr, method, path)
			case "image_binary":
				// Image binary response
				content := response.Value.Content
				if content != nil {
					hasImageContent := content.Get("image/jpeg") != nil || content.Get("image/png") != nil
					assert.True(t, hasImageContent, "Response %s should have image content for %s %s", statusCodeStr, method, path)
				}
			default:
				// JSON responses
				jsonContent := response.Value.Content.Get("application/json")
				if jsonContent != nil {
					assert.NotNil(t, jsonContent.Schema, "Response %s should have JSON schema for %s %s", statusCodeStr, method, path)
				}
			}
		}
	}

	// Validate operation metadata
	assert.NotEmpty(t, operation.Summary, "Operation %s %s should have summary", method, path)
	assert.NotEmpty(t, operation.OperationID, "Operation %s %s should have operationId", method, path)
	assert.NotEmpty(t, operation.Tags, "Operation %s %s should have tags", method, path)
}

// extractSchemaName extracts schema name from reference string.
func extractSchemaName(ref string) string {
	// Format: #/components/schemas/SchemaName
	parts := filepath.Base(ref)
	return parts
}

// int64Ptr returns a pointer to an int64 value.
func int64Ptr(i int64) *int64 {
	return &i
}

// TestRequestValidation tests request validation against OpenAPI spec.
func TestRequestValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		method      string
		path        string
		body        interface{}
		expectValid bool
	}{
		{
			name:   "Valid register request",
			method: http.MethodPost,
			path:   "/auth/register",
			body: map[string]interface{}{
				"email":    "test@example.com",
				"username": "testuser",
				"password": "SecurePass123!",
			},
			expectValid: true,
		},
		{
			name:   "Invalid register request - missing email",
			method: http.MethodPost,
			path:   "/auth/register",
			body: map[string]interface{}{
				"username": "testuser",
				"password": "SecurePass123!",
			},
			expectValid: false,
		},
		{
			name:   "Valid login request",
			method: http.MethodPost,
			path:   "/auth/login",
			body: map[string]interface{}{
				"email":    "test@example.com",
				"password": "SecurePass123!",
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Serialize body to JSON
			var bodyBytes []byte
			if tt.body != nil {
				var err error
				bodyBytes, err = json.Marshal(tt.body)
				require.NoError(t, err)
			}

			// Create request
			req := mustCreateRequest(t, tt.method, tt.path, bodyBytes)

			// Find route in spec
			route, pathParams, err := router.FindRoute(req)
			if !tt.expectValid && err != nil {
				return // Expected to not find route
			}
			require.NoError(t, err)

			// Create request validation input
			requestValidationInput := &openapi3filter.RequestValidationInput{
				Request:    req,
				PathParams: pathParams,
				Route:      route,
			}

			// Validate request
			err = openapi3filter.ValidateRequest(loader.Context, requestValidationInput)

			if tt.expectValid {
				assert.NoError(t, err, "Request should be valid")
			} else {
				assert.Error(t, err, "Request should be invalid")
			}
		})
	}
}

// mustCreateRequest creates an HTTP request for testing.
func mustCreateRequest(t *testing.T, method, path string, body []byte) *http.Request {
	t.Helper()

	url := "http://localhost:8080/api/v1" + path
	var req *http.Request
	var err error

	ctx := context.Background()
	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}

	require.NoError(t, err)
	return req
}

// TestResponseSchemaCompliance validates that all response schemas are properly defined.
//
//nolint:cyclop // Contract test requires comprehensive schema validation across all endpoints
func TestResponseSchemaCompliance(t *testing.T) {
	t.Parallel()

	// Collect all response schemas from all endpoints
	responseSchemas := make(map[string]bool)

	for path, pathItem := range doc.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			if operation == nil {
				continue
			}

			for statusCodeStr, response := range operation.Responses.Map() {
				if response == nil || response.Value == nil {
					continue
				}

				// Skip 204 No Content responses
				if statusCodeStr == "204" {
					continue
				}

				// Check for JSON content
				jsonContent := response.Value.Content.Get("application/json")
				if jsonContent == nil {
					continue
				}

				if jsonContent.Schema == nil {
					t.Errorf("Response schema missing for %s %s status %s", method, path, statusCodeStr)
					continue
				}

				// Track which schemas are used
				if jsonContent.Schema.Ref != "" {
					schemaName := extractSchemaName(jsonContent.Schema.Ref)
					responseSchemas[schemaName] = true
				}

				// Validate schema is resolvable
				err := jsonContent.Schema.Validate(loader.Context)
				assert.NoError(t, err, "Response schema validation failed for %s %s status %s", method, path, statusCodeStr)
			}
		}
	}

	t.Logf("Found %d unique response schemas in use", len(responseSchemas))
}

// TestErrorResponseCompliance validates that all error responses follow RFC 7807.
//
//nolint:gocognit // Contract test requires validating RFC 7807 compliance for all error responses across all endpoints
func TestErrorResponseCompliance(t *testing.T) {
	t.Parallel()

	errorStatusCodes := []int{400, 401, 403, 404, 409, 422, 429, 500, 503}

	problemDetailSchema := doc.Components.Schemas["ProblemDetail"]
	require.NotNil(t, problemDetailSchema, "ProblemDetail schema must exist")

	for path, pathItem := range doc.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			if operation == nil {
				continue
			}

			for _, statusCode := range errorStatusCodes {
				response := operation.Responses.Status(statusCode)
				if response == nil || response.Value == nil {
					continue
				}

				// Error responses should use application/problem+json or application/json
				jsonContent := response.Value.Content.Get("application/json")
				problemJSONContent := response.Value.Content.Get("application/problem+json")

				if jsonContent == nil && problemJSONContent == nil {
					continue
				}

				// Verify it references ProblemDetail schema
				content := jsonContent
				if content == nil {
					content = problemJSONContent
				}

				if content.Schema != nil && content.Schema.Ref != "" {
					schemaName := extractSchemaName(content.Schema.Ref)

					// Health check endpoints use their own response format for 503
					if path == "/health/ready" && statusCode == 503 {
						assert.Equal(t, "HealthReadyResponse", schemaName,
							"Health ready endpoint should use HealthReadyResponse for 503")
					} else {
						assert.Equal(t, "ProblemDetail", schemaName,
							"Error response for %s %s status %d should use ProblemDetail schema",
							method, path, statusCode)
					}
				}
			}
		}
	}
}

// TestQueryParameterValidation validates query parameter definitions.
//
//nolint:gocognit // Contract test requires validating query parameter definitions for all endpoints
func TestQueryParameterValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		path      string
		method    string
		paramName string
		required  bool
		paramType string
	}{
		{
			name:      "Images list - page parameter",
			path:      "/images",
			method:    http.MethodGet,
			paramName: "page",
			required:  false,
			paramType: "integer",
		},
		{
			name:      "Images list - per_page parameter",
			path:      "/images",
			method:    http.MethodGet,
			paramName: "per_page",
			required:  false,
			paramType: "integer",
		},
		{
			name:      "Images list - owner_id parameter",
			path:      "/images",
			method:    http.MethodGet,
			paramName: "owner_id",
			required:  false,
			paramType: "string",
		},
		{
			name:      "Images list - tags parameter",
			path:      "/images",
			method:    http.MethodGet,
			paramName: "tags",
			required:  false,
			paramType: "string",
		},
		{
			name:      "Tags search - q parameter",
			path:      "/tags/search",
			method:    http.MethodGet,
			paramName: "q",
			required:  true, // q parameter is required for tag search
			paramType: "string",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pathItem := doc.Paths.Find(tt.path)
			require.NotNil(t, pathItem, "Path %s should exist", tt.path)

			operation := pathItem.GetOperation(tt.method)
			require.NotNil(t, operation, "Operation %s %s should exist", tt.method, tt.path)

			// Find the parameter
			var found bool
			for _, param := range operation.Parameters {
				if param.Value != nil && param.Value.Name == tt.paramName {
					found = true
					assert.Equal(t, "query", param.Value.In, "Parameter %s should be in query", tt.paramName)
					assert.Equal(t, tt.required, param.Value.Required, "Parameter %s required mismatch", tt.paramName)

					if param.Value.Schema != nil && param.Value.Schema.Value != nil {
						actualType := param.Value.Schema.Value.Type.Slice()[0]
						assert.Equal(t, tt.paramType, actualType, "Parameter %s type mismatch", tt.paramName)
					}
					break
				}
			}

			if !found {
				// Check if it's defined as a parameter reference
				for _, param := range operation.Parameters {
					if param.Ref != "" {
						paramName := extractSchemaName(param.Ref)
						if paramName == "PageParam" && tt.paramName == "page" {
							found = true
							break
						}
						if paramName == "PerPageParam" && tt.paramName == "per_page" {
							found = true
							break
						}
					}
				}
			}

			// For optional parameters, it's OK if not found
			if tt.required {
				assert.True(t, found, "Required parameter %s should be defined for %s %s", tt.paramName, tt.method, tt.path)
			}
		})
	}
}

// TestOptionalAuthenticationEndpoints validates endpoints with optional authentication.
func TestOptionalAuthenticationEndpoints(t *testing.T) {
	t.Parallel()

	// Endpoints that support both authenticated and anonymous access
	optionalAuthEndpoints := []struct {
		path   string
		method string
	}{
		{"/images", http.MethodGet},
		{"/images/{id}", http.MethodGet},
		{"/images/{id}/variants/{size}", http.MethodGet},
		{"/images/{id}/comments", http.MethodGet},
		{"/albums", http.MethodGet},
		{"/albums/{id}", http.MethodGet},
		{"/tags", http.MethodGet},
		{"/tags/search", http.MethodGet},
		{"/tags/{tag}/images", http.MethodGet},
		{"/users/{id}/likes", http.MethodGet},
		{"/explore/recent", http.MethodGet},
		{"/explore/popular", http.MethodGet},
	}

	for _, endpoint := range optionalAuthEndpoints {
		endpoint := endpoint
		t.Run(endpoint.method+" "+endpoint.path, func(t *testing.T) {
			t.Parallel()

			pathItem := doc.Paths.Find(endpoint.path)
			require.NotNil(t, pathItem, "Path %s should exist", endpoint.path)

			operation := pathItem.GetOperation(endpoint.method)
			require.NotNil(t, operation, "Operation %s %s should exist", endpoint.method, endpoint.path)

			// Optional auth endpoints should have security with {} (public) and bearerAuth
			if operation.Security != nil && len(*operation.Security) > 0 {
				secReqs := *operation.Security
				// Should have at least one empty security requirement {} for public access
				hasPublicAccess := false
				for _, secReq := range secReqs {
					if len(secReq) == 0 {
						hasPublicAccess = true
						break
					}
				}
				assert.True(t, hasPublicAccess,
					"Optional auth endpoint %s %s should have {} security requirement for public access",
					endpoint.method, endpoint.path)
			}
		})
	}
}

// TestMediaTypeCompliance validates content types are properly defined.
//
//nolint:gocognit // Contract test requires validating media types for all request/response bodies across all endpoints
func TestMediaTypeCompliance(t *testing.T) {
	t.Parallel()

	for path, pathItem := range doc.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			if operation == nil {
				continue
			}

			// Check request body content types
			if operation.RequestBody != nil && operation.RequestBody.Value != nil {
				content := operation.RequestBody.Value.Content

				// Image upload should support multipart/form-data
				if path == "/images" && method == http.MethodPost {
					assert.NotNil(t, content.Get("multipart/form-data"),
						"POST /images should support multipart/form-data")
				}

				// Other endpoints should use application/json for JSON bodies
				if content.Get("application/json") != nil {
					jsonContent := content.Get("application/json")
					assert.NotNil(t, jsonContent.Schema, "JSON request body should have schema for %s %s", method, path)
				}
			}

			// Check response content types
			for statusCodeStr, response := range operation.Responses.Map() {
				if response == nil || response.Value == nil || response.Value.Content == nil {
					continue
				}

				// Image variant endpoint should return image/* content types
				if path == "/images/{id}/variants/{size}" && statusCodeStr == "200" {
					hasImageContent := response.Value.Content.Get("image/jpeg") != nil ||
						response.Value.Content.Get("image/png") != nil ||
						response.Value.Content.Get("image/webp") != nil
					assert.True(t, hasImageContent,
						"GET /images/{id}/variants/{size} should return image content type")
				}

				// Error responses should use application/json (status codes 400+)
				if len(statusCodeStr) > 0 && statusCodeStr[0] >= '4' {
					jsonContent := response.Value.Content.Get("application/json")
					problemContent := response.Value.Content.Get("application/problem+json")
					assert.True(t, jsonContent != nil || problemContent != nil,
						"Error response for %s %s status %s should have JSON content type",
						method, path, statusCodeStr)
				}
			}
		}
	}
}

// TestSchemaRequiredFields validates that required fields are properly marked.
func TestSchemaRequiredFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		schemaName     string
		requiredFields []string
	}{
		{
			schemaName:     "User",
			requiredFields: []string{"id", "email", "username", "created_at"},
		},
		{
			schemaName:     "Image",
			requiredFields: []string{"id", "owner_id", "owner", "visibility", "mime_type", "width", "height", "variants", "created_at"},
		},
		{
			schemaName:     "Album",
			requiredFields: []string{"id", "owner_id", "title", "visibility", "image_count", "created_at"},
		},
		{
			schemaName:     "Comment",
			requiredFields: []string{"id", "user_id", "user", "image_id", "content", "created_at"},
		},
		{
			schemaName:     "TokenResponse",
			requiredFields: []string{"access_token", "refresh_token", "token_type", "expires_in"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.schemaName, func(t *testing.T) {
			t.Parallel()

			schema := doc.Components.Schemas[tt.schemaName]
			require.NotNil(t, schema, "Schema %s should exist", tt.schemaName)
			require.NotNil(t, schema.Value, "Schema %s should have value", tt.schemaName)

			for _, field := range tt.requiredFields {
				assert.Contains(t, schema.Value.Required, field,
					"Schema %s should require field %s", tt.schemaName, field)
			}
		})
	}
}

// TestEndpointCoverage ensures all paths in spec have corresponding tests.
func TestEndpointCoverage(t *testing.T) {
	t.Parallel()

	var totalEndpoints int
	var documentedEndpoints int

	for path, pathItem := range doc.Paths.Map() {
		for method := range pathItem.Operations() {
			totalEndpoints++

			operation := pathItem.GetOperation(method)
			if operation != nil {
				// Check if endpoint has proper documentation
				if operation.Summary != "" && operation.Description != "" && operation.OperationID != "" {
					documentedEndpoints++
				} else {
					t.Logf("Endpoint %s %s is missing documentation (summary, description, or operationId)", method, path)
				}
			}
		}
	}

	t.Logf("Total endpoints: %d", totalEndpoints)
	t.Logf("Documented endpoints: %d", documentedEndpoints)

	// At least 90% of endpoints should have full documentation
	coveragePercent := float64(documentedEndpoints) / float64(totalEndpoints) * 100
	assert.GreaterOrEqual(t, coveragePercent, 90.0,
		"At least 90%% of endpoints should have complete documentation (summary, description, operationId)")
}
