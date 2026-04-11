package httpx_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type autoUsersEndpoint struct{}

type autoListUsersOutput struct {
	Body struct {
		Items []string `json:"items"`
	} `json:"body"`
}

type autoGetUserInput struct {
	ID int `path:"id"`
}

type autoGetUserOutput struct {
	Body struct {
		ID int `json:"id"`
	} `json:"body"`
}

type autoCreateUserInput struct {
	Body struct {
		Name string `json:"name"`
	} `json:"body"`
}

type autoCreateUserOutput struct {
	Body struct {
		Name string `json:"name"`
	} `json:"body"`
}

type autoProfileEndpoint struct{}
type autoAdvancedEndpoint struct{}

type autoProfileInput struct {
	ID int `path:"id"`
}

type autoAdvancedInput struct {
	TenantID  int `path:"tenant_id"`
	ProfileID int `path:"profile_id"`
}

type autoProfileOutput struct {
	Body struct {
		Name string `json:"name"`
	} `json:"body"`
}

func (e *autoUsersEndpoint) EndpointSpec() EndpointSpec {
	return EndpointSpec{
		Prefix:        "/api/v1/users",
		Tags:          Tags("users"),
		SummaryPrefix: "Users",
	}
}

func (e *autoUsersEndpoint) Register(registrar Registrar) {
	MustAuto(registrar,
		Auto(e.List),
		Auto(e.GetByID),
		Auto(e.Create),
	)
}

func (e *autoUsersEndpoint) List(_ context.Context, _ *struct{}) (*autoListUsersOutput, error) {
	out := &autoListUsersOutput{}
	out.Body.Items = []string{"Alice", "Bob"}
	return out, nil
}

func (e *autoUsersEndpoint) GetByID(_ context.Context, input *autoGetUserInput) (*autoGetUserOutput, error) {
	out := &autoGetUserOutput{}
	out.Body.ID = input.ID
	return out, nil
}

func (e *autoUsersEndpoint) Create(_ context.Context, input *autoCreateUserInput) (*autoCreateUserOutput, error) {
	out := &autoCreateUserOutput{}
	out.Body.Name = input.Body.Name
	return out, nil
}

func (e *autoProfileEndpoint) EndpointSpec() EndpointSpec {
	return EndpointSpec{Prefix: "/api/v1"}
}

func (e *autoProfileEndpoint) Register(registrar Registrar) {
	MustAuto(registrar,
		Auto(e.ListProfiles),
		Auto(e.GetProfileByID),
	)
}

func (e *autoProfileEndpoint) ListProfiles(_ context.Context, _ *struct{}) (*autoListUsersOutput, error) {
	out := &autoListUsersOutput{}
	out.Body.Items = []string{"primary"}
	return out, nil
}

func (e *autoProfileEndpoint) GetProfileByID(_ context.Context, input *autoProfileInput) (*autoProfileOutput, error) {
	out := &autoProfileOutput{}
	out.Body.Name = "profile"
	if input.ID > 0 {
		out.Body.Name = "profile-id"
	}
	return out, nil
}

func (e *autoAdvancedEndpoint) EndpointSpec() EndpointSpec {
	return EndpointSpec{Prefix: "/api/v1"}
}

func (e *autoAdvancedEndpoint) Register(registrar Registrar) {
	MustAuto(registrar,
		Auto(e.GetNearbyStoreByID),
		Auto(e.GetProfileByTenantIDAndProfileID),
	)
}

func (e *autoAdvancedEndpoint) GetNearbyStoreByID(_ context.Context, input *autoProfileInput) (*autoProfileOutput, error) {
	out := &autoProfileOutput{}
	out.Body.Name = "nearby-store"
	if input.ID > 0 {
		out.Body.Name = "nearby-store-id"
	}
	return out, nil
}

func (e *autoAdvancedEndpoint) GetProfileByTenantIDAndProfileID(_ context.Context, input *autoAdvancedInput) (*autoProfileOutput, error) {
	out := &autoProfileOutput{}
	out.Body.Name = "profile"
	if input.TenantID > 0 && input.ProfileID > 0 {
		out.Body.Name = "profile-scoped"
	}
	return out, nil
}

func TestServer_RegisterOnly_AutoEndpointRoutes(t *testing.T) {
	server := newServer()

	server.RegisterOnly(&autoUsersEndpoint{})

	req := newTestRequest(http.MethodGet, "/api/v1/users/42", nil)
	rec := serveRequest(t, server, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"id":42`)
	assert.True(t, server.HasRoute(http.MethodGet, "/api/v1/users"))
	assert.True(t, server.HasRoute(http.MethodGet, "/api/v1/users/{id}"))
	assert.True(t, server.HasRoute(http.MethodPost, "/api/v1/users"))

	pathItem := server.OpenAPI().Paths["/api/v1/users/{id}"]
	if assert.NotNil(t, pathItem) && assert.NotNil(t, pathItem.Get) {
		assert.Equal(t, "Users Get", pathItem.Get.Summary)
		assert.Contains(t, pathItem.Get.Tags, "users")
	}
}

func TestServer_RegisterOnly_AutoEndpointNamedResources(t *testing.T) {
	server := newServer()

	server.RegisterOnly(&autoProfileEndpoint{})

	assert.True(t, server.HasRoute(http.MethodGet, "/api/v1/profiles"))
	assert.True(t, server.HasRoute(http.MethodGet, "/api/v1/profile/{id}"))

	listPathItem := server.OpenAPI().Paths["/api/v1/profiles"]
	if assert.NotNil(t, listPathItem) && assert.NotNil(t, listPathItem.Get) {
		assert.Equal(t, "List Profiles", listPathItem.Get.Summary)
	}

	getPathItem := server.OpenAPI().Paths["/api/v1/profile/{id}"]
	if assert.NotNil(t, getPathItem) && assert.NotNil(t, getPathItem.Get) {
		assert.Equal(t, "Get Profile", getPathItem.Get.Summary)
	}
}

func TestRegisterAuto_InvalidHandlerName(t *testing.T) {
	server := newServer()

	err := RegisterAuto(server.Group("/api/v1/users"),
		Auto(func(_ context.Context, _ *struct{}) (*autoProfileOutput, error) {
			return &autoProfileOutput{}, nil
		}),
	)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidHandlerName))
}

func TestServer_RegisterOnly_AutoEndpointTokenAwareParsing(t *testing.T) {
	server := newServer()

	server.RegisterOnly(&autoAdvancedEndpoint{})

	assert.True(t, server.HasRoute(http.MethodGet, "/api/v1/nearby-store/{id}"))
	assert.True(t, server.HasRoute(http.MethodGet, "/api/v1/profile/{tenant-id}/{profile-id}"))

	nearbyPathItem := server.OpenAPI().Paths["/api/v1/nearby-store/{id}"]
	if assert.NotNil(t, nearbyPathItem) && assert.NotNil(t, nearbyPathItem.Get) {
		assert.Equal(t, "Get Nearby Store", nearbyPathItem.Get.Summary)
	}

	profilePathItem := server.OpenAPI().Paths["/api/v1/profile/{tenant-id}/{profile-id}"]
	if assert.NotNil(t, profilePathItem) && assert.NotNil(t, profilePathItem.Get) {
		assert.Equal(t, "Get Profile", profilePathItem.Get.Summary)
	}
}
