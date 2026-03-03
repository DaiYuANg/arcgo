---
sidebar_position: 4
---

# Huma OpenAPI Integration

`httpx` integrates [Huma v2](https://github.com/danielgtaylor/huma) for automatic OpenAPI generation.

## Enable OpenAPI

### Basic setup

```go
ginAdapter := gin.New()

ginAdapter.WithHuma(httpx.ToAdapterHumaOptions(httpx.HumaOptions{
    Enabled:     true,
    Title:       "My API",
    Version:     "1.0.0",
    Description: "My API Documentation",
}))
```

### Full options

```go
ginAdapter.WithHuma(httpx.ToAdapterHumaOptions(httpx.HumaOptions{
    Enabled:         true,
    Title:           "My API",
    Version:         "1.0.0",
    Description:     "My API Documentation",
    TermsOfService:  "https://example.com/terms",
    ContactName:     "API Support",
    ContactEmail:    "support@example.com",
    ContactURL:      "https://example.com/support",
    LicenseName:     "MIT",
    LicenseURL:      "https://opensource.org/licenses/MIT",
    DocsPath:        "/docs",
    OpenAPIPath:     "/openapi.json",
    Servers: []httpx.HumaServer{
        {URL: "https://api.example.com", Description: "Production"},
        {URL: "https://staging-api.example.com", Description: "Staging"},
    },
}))
```

## Docs endpoints

- OpenAPI JSON: `http://localhost:8080/openapi.json`
- Swagger UI: `http://localhost:8080/docs`
- RapiDoc: `http://localhost:8080/docs/rapidoc`
- Elements: `http://localhost:8080/docs/elements`

## Endpoint annotations

```go
// ListUsers
// @Summary List users
// @Description Get a list of all users
// @Tags users
// @OperationID listUsers
// @Success 200 {body} []User
func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    // ...
}
```

```go
// GetUser
// @Param id path string true "User ID"
// @Success 200 {body} User
// @Failure 404 {body} Error
func (e *UserEndpoint) GetUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    // ...
}
```

## Model definitions

```go
// @Model
type User struct {
    ID       int       `json:"id" example:"1"`
    Username string    `json:"username" example:"john_doe"`
    Email    string    `json:"email" example:"john@example.com"`
    CreatedAt time.Time `json:"created_at"`
}

// @Model
type CreateUserInput struct {
    Username string `json:"username" validate:"required,min=3,max=50"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}
```

## Security schemes

### Bearer token

```go
SecuritySchemes: map[string]interface{}{
    "bearerAuth": map[string]interface{}{
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT",
    },
}
```

### API key

```go
SecuritySchemes: map[string]interface{}{
    "apiKeyAuth": map[string]interface{}{
        "type": "apiKey",
        "in": "header",
        "name": "X-API-Key",
    },
}
```

After startup, open `http://localhost:8080/docs` to view docs.
