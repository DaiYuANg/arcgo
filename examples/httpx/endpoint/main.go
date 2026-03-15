package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/DaiYuANg/archgo/httpx"
	"github.com/DaiYuANg/archgo/httpx/adapter"
	"github.com/DaiYuANg/archgo/httpx/adapter/std"
	"github.com/DaiYuANg/archgo/httpx/examples/shared"
	"github.com/DaiYuANg/archgo/pkg/randomport"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
)

// ==================== User Endpoint ====================

type UserEndpoint struct {
	httpx.BaseEndpoint
}

type createUserInput struct {
	Body struct {
		Name  string `json:"name" validate:"required,min=2,max=64"`
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

func (e *UserEndpoint) RegisterRoutes(server httpx.ServerRuntime) {
	api := server.Group("/api/v1/users")

	httpx.MustGroupGet(api, "", func(ctx context.Context, input *struct{}) (*listUsersOutput, error) {
		out := &listUsersOutput{}
		out.Body.Users = []string{"Alice", "Bob", "Charlie"}
		return out, nil
	})

	httpx.MustGroupGet(api, "/{id}", func(ctx context.Context, input *getUserInput) (*getUserOutput, error) {
		out := &getUserOutput{}
		out.Body.ID = input.ID
		out.Body.Name = "User-" + fmt.Sprint(input.ID)
		return out, nil
	})

	httpx.MustGroupPost(api, "", func(ctx context.Context, input *createUserInput) (*createUserOutput, error) {
		out := &createUserOutput{}
		out.Body.ID = 1001
		out.Body.Name = input.Body.Name
		out.Body.Email = input.Body.Email
		return out, nil
	})
}

// ==================== Health Endpoint ====================

type HealthEndpoint struct {
	httpx.BaseEndpoint
}

type healthOutput struct {
	Body struct {
		Status string `json:"status"`
	} `json:"body"`
}

func (e *HealthEndpoint) RegisterRoutes(server httpx.ServerRuntime) {
	httpx.MustGet(server, "/health", func(ctx context.Context, input *struct{}) (*healthOutput, error) {
		out := &healthOutput{}
		out.Body.Status = "ok"
		return out, nil
	})
}

// ==================== Order Endpoint (with hooks) ====================

type OrderEndpoint struct {
	httpx.BaseEndpoint
}

type createOrderInput struct {
	Body struct {
		ProductID int `json:"product_id" validate:"required,min=1"`
		Quantity  int `json:"quantity" validate:"required,min=1"`
	} `json:"body"`
}

type createOrderOutput struct {
	Body struct {
		OrderID   int `json:"order_id"`
		ProductID int `json:"product_id"`
		Quantity  int `json:"quantity"`
	} `json:"body"`
}

func (e *OrderEndpoint) RegisterRoutes(server httpx.ServerRuntime) {
	api := server.Group("/api/v1/orders")

	httpx.MustGroupPost(api, "", func(ctx context.Context, input *createOrderInput) (*createOrderOutput, error) {
		out := &createOrderOutput{}
		out.Body.OrderID = 5001
		out.Body.ProductID = input.Body.ProductID
		out.Body.Quantity = input.Body.Quantity
		return out, nil
	})
}

func main() {
	logger, closeLogger, err := shared.NewLogger()
	if err != nil {
		panic(err)
	}
	defer closeLogger()

	stdAdapter := std.New(adapter.HumaOptions{
		Title:       "Endpoint Example API",
		Version:     "1.0.0",
		Description: "Endpoint pattern example built with httpx",
		DocsPath:    "/docs",
		OpenAPIPath: "/openapi.json",
	})
	stdAdapter.Router().Use(middleware.Logger, middleware.Recoverer)

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

	if err := server.ListenAndServe(addr); err != nil {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
