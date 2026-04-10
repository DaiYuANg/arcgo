// Package main demonstrates the httpx endpoint registration pattern.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/DaiYuANg/arcgo/examples/httpx/shared"
	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/pkg/randomport"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
)

// ==================== User Endpoint ====================

// UserEndpoint registers demo user routes.
type UserEndpoint struct {
	httpx.BaseEndpoint
}

type createUserInput struct {
	Body struct {
		Name  string `json:"name"  validate:"required,min=2,max=64"`
		Email string `json:"email" validate:"required,email"`
	} `json:"body"`
}

type createUserOutput struct {
	Body struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"body"`
}

type getUserInput struct {
	ID int `path:"id"`
}

type getUserOutput struct {
	Body struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"body"`
}

type listUsersOutput struct {
	Body struct {
		Users []string `json:"users"`
	} `json:"body"`
}

// RegisterRoutes registers the user routes on the server.
func (e *UserEndpoint) EndpointSpec() httpx.EndpointSpec {
	return httpx.EndpointSpec{
		Prefix:        "/api/v1/users",
		Tags:          []string{"users", "v1"},
		SummaryPrefix: "Users",
		Description:   "User management endpoints",
	}
}

// RegisterGroupRoutes registers the user routes on the endpoint group.
func (e *UserEndpoint) RegisterGroupRoutes(api *httpx.Group) {
	httpx.MustGroupGet(api, "", func(_ context.Context, _ *struct{}) (*listUsersOutput, error) {
		out := &listUsersOutput{}
		out.Body.Users = []string{"Alice", "Bob", "Charlie"}
		return out, nil
	}, func(op *huma.Operation) {
		op.Summary = "List"
	})

	httpx.MustGroupGet(api, "/{id}", func(_ context.Context, input *getUserInput) (*getUserOutput, error) {
		out := &getUserOutput{}
		out.Body.ID = input.ID
		out.Body.Name = "User-" + strconv.Itoa(input.ID)
		return out, nil
	}, func(op *huma.Operation) {
		op.Summary = "Get"
	})

	httpx.MustGroupPost(api, "", func(_ context.Context, input *createUserInput) (*createUserOutput, error) {
		out := &createUserOutput{}
		out.Body.ID = 1001
		out.Body.Name = input.Body.Name
		out.Body.Email = input.Body.Email
		return out, nil
	}, func(op *huma.Operation) {
		op.Summary = "Create"
	})
}

// ==================== Health Endpoint ====================

// HealthEndpoint registers the health check route.
type HealthEndpoint struct {
	httpx.BaseEndpoint
}

type healthOutput struct {
	Body struct {
		Status string `json:"status"`
	} `json:"body"`
}

// RegisterRoutes registers the health route on the server.
func (e *HealthEndpoint) RegisterRoutes(server httpx.ServerRuntime) {
	httpx.MustGet(server, "/health", func(_ context.Context, _ *struct{}) (*healthOutput, error) {
		out := &healthOutput{}
		out.Body.Status = "ok"
		return out, nil
	})
}

// ==================== Order Endpoint (with hooks) ====================

// OrderEndpoint registers demo order routes.
type OrderEndpoint struct {
	httpx.BaseEndpoint
}

type createOrderInput struct {
	Body struct {
		ProductID int `json:"product_id" validate:"required,min=1"`
		Quantity  int `json:"quantity"   validate:"required,min=1"`
	} `json:"body"`
}

type createOrderOutput struct {
	Body struct {
		OrderID   int `json:"order_id"`
		ProductID int `json:"product_id"`
		Quantity  int `json:"quantity"`
	} `json:"body"`
}

func (e *OrderEndpoint) EndpointSpec() httpx.EndpointSpec {
	return httpx.EndpointSpec{
		Prefix:        "/api/v1/orders",
		Tags:          []string{"orders", "v1"},
		SummaryPrefix: "Orders",
		Description:   "Order management endpoints",
	}
}

// RegisterGroupRoutes registers the order routes on the endpoint group.
func (e *OrderEndpoint) RegisterGroupRoutes(api *httpx.Group) {
	httpx.MustGroupPost(api, "", func(_ context.Context, input *createOrderInput) (*createOrderOutput, error) {
		out := &createOrderOutput{}
		out.Body.OrderID = 5001
		out.Body.ProductID = input.Body.ProductID
		out.Body.Quantity = input.Body.Quantity
		return out, nil
	}, func(op *huma.Operation) {
		op.Summary = "Create"
	})
}

func main() {
	logger, closeLogger, err := shared.NewLogger()
	if err != nil {
		panic(err)
	}

	router := chi.NewMux()
	router.Use(middleware.Logger, middleware.Recoverer)

	stdAdapter := std.New(router, adapter.HumaOptions{
		Title:       "Endpoint Example API",
		Version:     "1.0.0",
		Description: "Endpoint pattern example built with httpx",
		DocsPath:    "/docs",
		OpenAPIPath: "/openapi.json",
	})

	server := httpx.New(
		httpx.WithAdapter(stdAdapter),
		httpx.WithBasePath("/"),
		httpx.WithValidation(),
		httpx.WithPrintRoutes(true),
		httpx.WithValidator(validator.New(validator.WithRequiredStructEnabled())),
	)

	// 方式 1: 使用 RegisterOnly 批量注册（无 hook）
	server.RegisterOnly(
		&HealthEndpoint{},
		&UserEndpoint{},
		&OrderEndpoint{},
	)

	// 方式 2: 使用 Register 带 hook 注册单个 endpoint
	// server.Register(&HealthEndpoint{})
	// server.Register(&UserEndpoint{}, func(s *httpx.Server, e httpx.Endpoint) {
	// 	fmt.Println("Registering UserEndpoint...")
	// })
	// server.Register(&OrderEndpoint{},
	// 	func(s *httpx.Server, e httpx.Endpoint) {
	// 		fmt.Println("Before OrderEndpoint registration")
	// 	},
	// 	func(s *httpx.Server, e httpx.Endpoint) {
	// 		fmt.Println("After OrderEndpoint registration")
	// 	},
	// )

	port := randomport.MustFind()
	addr := fmt.Sprintf(":%d", port)
	logger.Info("example server starting",
		slog.String("example", "endpoint"),
		slog.String("address", addr),
		slog.String("openapi", fmt.Sprintf("http://localhost%s/openapi.json", addr)),
		slog.String("docs", fmt.Sprintf("http://localhost%s/docs", addr)),
	)

	if err := server.ListenPort(port); err != nil {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		closeLogger()
		os.Exit(1)
	}
	closeLogger()
}
