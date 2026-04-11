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
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
)

// ==================== User Endpoint ====================

// UserEndpoint registers demo user routes.
type UserEndpoint struct{}

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

func (e *UserEndpoint) EndpointSpec() httpx.EndpointSpec {
	return httpx.EndpointSpec{
		Prefix:        "/api/v1/users",
		Tags:          httpx.Tags("users", "v1"),
		SummaryPrefix: "Users",
		Description:   "User management endpoints",
	}
}

// Register registers the user routes on the endpoint scope.
func (e *UserEndpoint) Register(registrar httpx.Registrar) {
	httpx.MustAuto(registrar,
		httpx.Auto(e.List),
		httpx.Auto(e.GetByID),
		httpx.Auto(e.Create),
	)
}

func (e *UserEndpoint) List(_ context.Context, _ *struct{}) (*listUsersOutput, error) {
	out := &listUsersOutput{}
	out.Body.Users = []string{"Alice", "Bob", "Charlie"}
	return out, nil
}

func (e *UserEndpoint) GetByID(_ context.Context, input *getUserInput) (*getUserOutput, error) {
	out := &getUserOutput{}
	out.Body.ID = input.ID
	out.Body.Name = "User-" + strconv.Itoa(input.ID)
	return out, nil
}

func (e *UserEndpoint) Create(_ context.Context, input *createUserInput) (*createUserOutput, error) {
	out := &createUserOutput{}
	out.Body.ID = 1001
	out.Body.Name = input.Body.Name
	out.Body.Email = input.Body.Email
	return out, nil
}

// ==================== Health Endpoint ====================

// HealthEndpoint registers the health check route.
type HealthEndpoint struct{}

type healthOutput struct {
	Body struct {
		Status string `json:"status"`
	} `json:"body"`
}

// Register registers the health route on the root scope.
func (e *HealthEndpoint) Register(registrar httpx.Registrar) {
	httpx.MustAuto(registrar, httpx.Auto(e.GetHealth))
}

func (e *HealthEndpoint) GetHealth(_ context.Context, _ *struct{}) (*healthOutput, error) {
	out := &healthOutput{}
	out.Body.Status = "ok"
	return out, nil
}

// ==================== Order Endpoint (with hooks) ====================

// OrderEndpoint registers demo order routes.
type OrderEndpoint struct{}

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
		Tags:          httpx.Tags("orders", "v1"),
		SummaryPrefix: "Orders",
		Description:   "Order management endpoints",
	}
}

// Register registers the order routes on the endpoint scope.
func (e *OrderEndpoint) Register(registrar httpx.Registrar) {
	httpx.MustAuto(registrar, httpx.Auto(e.Create))
}

func (e *OrderEndpoint) Create(_ context.Context, input *createOrderInput) (*createOrderOutput, error) {
	out := &createOrderOutput{}
	out.Body.OrderID = 5001
	out.Body.ProductID = input.Body.ProductID
	out.Body.Quantity = input.Body.Quantity
	return out, nil
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
	// server.Register(&UserEndpoint{}, httpx.EndpointHooks{
	// 	Before: func(_ httpx.ServerRuntime, _ any) {
	// 		fmt.Println("Registering UserEndpoint...")
	// 	},
	// })
	// server.Register(&OrderEndpoint{},
	// 	httpx.EndpointHooks{Before: func(_ httpx.ServerRuntime, _ any) {
	// 		fmt.Println("Before OrderEndpoint registration")
	// 	}},
	// 	httpx.EndpointHooks{After: func(_ httpx.ServerRuntime, _ any) {
	// 		fmt.Println("After OrderEndpoint registration")
	// 	}},
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
