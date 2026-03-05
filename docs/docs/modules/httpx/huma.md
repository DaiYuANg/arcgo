---
sidebar_position: 4
---

# Huma OpenAPI 集成

`httpx` 集成了 [Huma v2](https://github.com/danielgtaylor/huma)，提供自动 OpenAPI 文档生成功能。

## 什么是 Huma？

Huma 是一个现代化的 Go Web 框架，提供：
- 🚀 自动 OpenAPI 3.0 文档生成
- 📝 自动请求/响应验证
- 🔒 类型安全
- ⚡ 高性能

## 启用 OpenAPI

### 基本配置

```go
package main

import (
    "github.com/DaiYuANg/arcgo/httpx"
    "github.com/DaiYuANg/arcgo/httpx/adapter/gin"
)

func main() {
    ginAdapter := gin.New()
    
    // 启用 Huma OpenAPI
    ginAdapter.WithHuma(httpx.ToAdapterHumaOptions(httpx.HumaOptions{
        Enabled:     true,
        Title:       "My API",
        Version:     "1.0.0",
        Description: "My API Documentation",
    }))
    
    server := httpx.NewServer(httpx.WithAdapter(ginAdapter))
    // ...
}
```

### 完整配置选项

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
    
    // 文档路径
    DocsPath:        "/docs",
    OpenAPIPath:     "/openapi.json",
    
    // 服务器信息
    Servers: []httpx.HumaServer{
        {
            URL:         "https://api.example.com",
            Description: "Production server",
        },
        {
            URL:         "https://staging-api.example.com",
            Description: "Staging server",
        },
    },
}))
```

## 访问文档

启用后，可以通过以下路径访问：

- **OpenAPI JSON**: `http://localhost:8080/openapi.json`
- **Swagger UI**: `http://localhost:8080/docs`
- **RapiDoc**: `http://localhost:8080/docs/rapidoc`
- **Elements**: `http://localhost:8080/docs/elements`

## 添加 OpenAPI 注解

### 基本注解

```go
type UserEndpoint struct {
    httpx.BaseEndpoint
}

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

### 参数注解

```go
// GetUser
// @Summary Get user by ID
// @Description Get a user by their ID
// @Tags users
// @OperationID getUser
// @Param id path string true "User ID"
// @Success 200 {body} User
// @Failure 404 {body} Error
func (e *UserEndpoint) GetUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    id := e.Param(w, r, "id")
    // ...
}
```

### 请求体注解

```go
// CreateUser
// @Summary Create a new user
// @Description Create a new user with the provided data
// @Tags users
// @OperationID createUser
// @Param user body CreateUserInput true "User data"
// @Success 201 {body} User
// @Failure 400 {body} Error
func (e *UserEndpoint) CreateUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    var input CreateUserInput
    if err := e.ParseJSON(r, &input); err != nil {
        return e.BadRequest(w, "Invalid input")
    }
    // ...
}
```

### 查询参数注解

```go
// ListUsers
// @Summary List users with pagination
// @Description Get a paginated list of users
// @Tags users
// @OperationID listUsers
// @Param page query int false "Page number" default(1)
// @Param size query int false "Page size" default(10)
// @Param search query string false "Search term"
// @Success 200 {body} []User
func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    page := e.QueryInt(w, r, "page", 1)
    size := e.QueryInt(w, r, "size", 10)
    search := e.Query(w, r, "search")
    // ...
}
```

## 定义模型

### 响应模型

```go
// User 用户模型
// @Model
type User struct {
    // 用户 ID
    ID       int    `json:"id" example:"1"`
    // 用户名
    Username string `json:"username" example:"john_doe"`
    // 邮箱
    Email    string `json:"email" example:"john@example.com"`
    // 创建时间
    CreatedAt time.Time `json:"created_at"`
}

// CreateUserInput 创建用户输入
// @Model
type CreateUserInput struct {
    // 用户名
    Username string `json:"username" validate:"required,min=3,max=50"`
    // 邮箱
    Email    string `json:"email" validate:"required,email"`
    // 密码
    Password string `json:"password" validate:"required,min=8"`
}

// Error 错误响应
// @Model
type Error struct {
    // 错误代码
    Code int `json:"code"`
    // 错误信息
    Message string `json:"message"`
}
```

## 安全认证

### Bearer Token

```go
// WithHuma 配置中添加安全方案
ginAdapter.WithHuma(httpx.ToAdapterHumaOptions(httpx.HumaOptions{
    Enabled: true,
    Title:   "My API",
    Version: "1.0.0",
    
    // 安全方案
    SecuritySchemes: map[string]interface{}{
        "bearerAuth": map[string]interface{}{
            "type": "http",
            "scheme": "bearer",
            "bearerFormat": "JWT",
        },
    },
}))

// 在 Endpoint 中使用
// ListUsers
// @Summary List users
// @Security bearerAuth
func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    // ...
}
```

### API Key

```go
ginAdapter.WithHuma(httpx.ToAdapterHumaOptions(httpx.HumaOptions{
    Enabled: true,
    
    SecuritySchemes: map[string]interface{}{
        "apiKeyAuth": map[string]interface{}{
            "type": "apiKey",
            "in": "header",
            "name": "X-API-Key",
        },
    },
}))

// ListUsers
// @Summary List users
// @Security apiKeyAuth
func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    // ...
}
```

## 自定义 OpenAPI

### 添加标签

```go
ginAdapter.WithHuma(httpx.ToAdapterHumaOptions(httpx.HumaOptions{
    Enabled: true,
    Title:   "My API",
    
    // 自定义标签
    Tags: []map[string]interface{}{
        {
            "name": "users",
            "description": "User management endpoints",
        },
        {
            "name": "products",
            "description": "Product management endpoints",
        },
    },
}))
```

### 添加服务器

```go
ginAdapter.WithHuma(httpx.ToAdapterHumaOptions(httpx.HumaOptions{
    Enabled: true,
    
    Servers: []httpx.HumaServer{
        {
            URL:         "https://api.example.com",
            Description: "Production server",
        },
        {
            URL:         "https://staging-api.example.com",
            Description: "Staging server",
        },
        {
            URL:         "http://localhost:8080",
            Description: "Development server",
        },
    },
}))
```

## 完整示例

```go
package main

import (
    "context"
    "net/http"
    "time"
    "github.com/DaiYuANg/arcgo/httpx"
    "github.com/DaiYuANg/arcgo/httpx/adapter/gin"
)

// User 用户模型
// @Model
type User struct {
    ID        int       `json:"id" example:"1"`
    Username  string    `json:"username" example:"john_doe"`
    Email     string    `json:"email" example:"john@example.com"`
    CreatedAt time.Time `json:"created_at"`
}

// CreateUserInput 创建用户输入
// @Model
type CreateUserInput struct {
    Username string `json:"username" validate:"required,min=3,max=50"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

// Error 错误响应
// @Model
type Error struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

type UserEndpoint struct {
    httpx.BaseEndpoint
}

// ListUsers
// @Summary List users
// @Description Get a list of all users
// @Tags users
// @OperationID listUsers
// @Success 200 {body} []User
func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    users := []User{
        {ID: 1, Username: "alice", Email: "alice@example.com"},
        {ID: 2, Username: "bob", Email: "bob@example.com"},
    }
    e.Success(w, users)
    return nil
}

// GetUser
// @Summary Get user by ID
// @Description Get a user by their ID
// @Tags users
// @OperationID getUser
// @Param id path string true "User ID"
// @Success 200 {body} User
// @Failure 404 {body} Error
func (e *UserEndpoint) GetUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    id := e.Param(w, r, "id")
    e.Success(w, User{
        ID: 1, Username: "alice", Email: "alice@example.com",
    })
    return nil
}

// CreateUser
// @Summary Create a new user
// @Description Create a new user with the provided data
// @Tags users
// @OperationID createUser
// @Param user body CreateUserInput true "User data"
// @Success 201 {body} User
// @Failure 400 {body} Error
func (e *UserEndpoint) CreateUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    var input CreateUserInput
    if err := e.ParseJSON(r, &input); err != nil {
        return e.BadRequest(w, "Invalid input")
    }
    e.SuccessWithCode(w, 201, User{
        ID: 1, Username: input.Username, Email: input.Email,
    })
    return nil
}

func main() {
    ginAdapter := gin.New()
    ginAdapter.Engine().Use(gin.Logger(), gin.Recovery())
    
    // 启用 Huma OpenAPI
    ginAdapter.WithHuma(httpx.ToAdapterHumaOptions(httpx.HumaOptions{
        Enabled:     true,
        Title:       "My API",
        Version:     "1.0.0",
        Description: "My API Documentation",
        DocsPath:    "/docs",
        OpenAPIPath: "/openapi.json",
    }))
    
    server := httpx.NewServer(
        httpx.WithAdapter(ginAdapter),
        httpx.WithBasePath("/api"),
    )
    
    _ = server.Register(&UserEndpoint{})
    server.ListenAndServe(":8080")
}
```

访问 `http://localhost:8080/docs` 查看 OpenAPI 文档。
