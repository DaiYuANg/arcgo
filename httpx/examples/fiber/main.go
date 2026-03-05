package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter/fiber"
	"github.com/DaiYuANg/arcgo/logx"
	"github.com/danielgtaylor/huma/v2"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
)

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserStore struct {
	mu     sync.RWMutex
	nextID int
	users  map[int]User
}

func NewUserStore() *UserStore {
	now := time.Now().UTC()
	return &UserStore{
		nextID: 3,
		users: map[int]User{
			1: {
				ID:        1,
				Name:      "Alice",
				Email:     "alice@example.com",
				Age:       26,
				CreatedAt: now,
				UpdatedAt: now,
			},
			2: {
				ID:        2,
				Name:      "Bob",
				Email:     "bob@example.com",
				Age:       30,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
}

func (s *UserStore) List(search string, limit, offset int) ([]User, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]User, 0, len(s.users))
	for _, u := range s.users {
		if search != "" && !strings.Contains(strings.ToLower(u.Name+u.Email), strings.ToLower(search)) {
			continue
		}
		items = append(items, u)
	}

	total := len(items)
	if offset >= total {
		return []User{}, total
	}

	end := offset + limit
	if end > total {
		end = total
	}
	return items[offset:end], total
}

func (s *UserStore) Get(id int) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	return u, ok
}

func (s *UserStore) Create(in CreateUserBody) User {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	u := User{
		ID:        s.nextID,
		Name:      in.Name,
		Email:     in.Email,
		Age:       in.Age,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.users[u.ID] = u
	s.nextID++
	return u
}

func (s *UserStore) Update(id int, in UpdateUserBody) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[id]
	if !ok {
		return User{}, false
	}

	if in.Name != nil {
		u.Name = *in.Name
	}
	if in.Email != nil {
		u.Email = *in.Email
	}
	if in.Age != nil {
		u.Age = *in.Age
	}
	u.UpdatedAt = time.Now().UTC()
	s.users[id] = u
	return u, true
}

func (s *UserStore) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[id]; !ok {
		return false
	}
	delete(s.users, id)
	return true
}

type ListUsersInput struct {
	Limit int    `query:"limit"`
	Page  int    `query:"page"`
	Q     string `query:"q"`
}

type ListUsersOutput struct {
	Body struct {
		Items []User `json:"items"`
		Total int    `json:"total"`
		Page  int    `json:"page"`
		Limit int    `json:"limit"`
	} `json:"body"`
}

type GetUserInput struct {
	ID int `path:"id"`
}

type GetUserOutput struct {
	Body User `json:"body"`
}

type CreateUserBody struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

type CreateUserInput struct {
	Body CreateUserBody `json:"body"`
}

type CreateUserOutput struct {
	Body User `json:"body"`
}

type UpdateUserBody struct {
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
	Age   *int    `json:"age,omitempty"`
}

type UpdateUserInput struct {
	ID   int            `path:"id"`
	Body UpdateUserBody `json:"body"`
}

type UpdateUserOutput struct {
	Body User `json:"body"`
}

type DeleteUserInput struct {
	ID int `path:"id"`
}

type DeleteUserOutput struct {
	Body struct {
		Deleted bool `json:"deleted"`
	} `json:"body"`
}

type HealthOutput struct {
	Body struct {
		Status string `json:"status"`
		Time   string `json:"time"`
	} `json:"body"`
}

func main() {
	logger, err := logx.New(logx.WithConsole(true), logx.WithDebugLevel())
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	store := NewUserStore()
	fiberAdapter := fiber.New()
	fiberAdapter.App().Use(fiberrecover.New(), fiberlogger.New())

	server := httpx.NewServer(
		httpx.WithAdapter(fiberAdapter),
		httpx.WithLogger(logx.NewSlog(logger)),
		httpx.WithPrintRoutes(true),
		httpx.WithHuma(httpx.HumaOptions{
			Enabled:     true,
			Title:       "ArcGo Fiber API",
			Version:     "1.0.0",
			Description: "Typed Fiber API example",
			DocsPath:    "/docs",
			OpenAPIPath: "/openapi.json",
		}),
	)

	if err = httpx.Get(server, "/health", func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
		out := &HealthOutput{}
		out.Body.Status = "ok"
		out.Body.Time = time.Now().UTC().Format(time.RFC3339)
		return out, nil
	}, huma.OperationTags("system")); err != nil {
		panic(err)
	}

	api := server.Group("/api/v1")

	if err = httpx.GroupGet(api, "/users", func(ctx context.Context, input *ListUsersInput) (*ListUsersOutput, error) {
		limit := input.Limit
		if limit <= 0 {
			limit = 10
		}
		if limit > 100 {
			limit = 100
		}

		page := input.Page
		if page <= 0 {
			page = 1
		}
		offset := (page - 1) * limit

		items, total := store.List(input.Q, limit, offset)
		out := &ListUsersOutput{}
		out.Body.Items = items
		out.Body.Total = total
		out.Body.Page = page
		out.Body.Limit = limit
		return out, nil
	}, huma.OperationTags("users")); err != nil {
		panic(err)
	}

	if err = httpx.GroupGet(api, "/users/{id}", func(ctx context.Context, input *GetUserInput) (*GetUserOutput, error) {
		u, ok := store.Get(input.ID)
		if !ok {
			return nil, httpx.NewError(404, "user not found")
		}
		out := &GetUserOutput{}
		out.Body = u
		return out, nil
	}, huma.OperationTags("users")); err != nil {
		panic(err)
	}

	if err = httpx.GroupPost(api, "/users", func(ctx context.Context, input *CreateUserInput) (*CreateUserOutput, error) {
		if strings.TrimSpace(input.Body.Name) == "" {
			return nil, httpx.NewError(400, "name is required")
		}
		if strings.TrimSpace(input.Body.Email) == "" {
			return nil, httpx.NewError(400, "email is required")
		}
		u := store.Create(input.Body)
		out := &CreateUserOutput{}
		out.Body = u
		return out, nil
	}, huma.OperationTags("users")); err != nil {
		panic(err)
	}

	if err = httpx.GroupPut(api, "/users/{id}", func(ctx context.Context, input *UpdateUserInput) (*UpdateUserOutput, error) {
		u, ok := store.Update(input.ID, input.Body)
		if !ok {
			return nil, httpx.NewError(404, "user not found")
		}
		out := &UpdateUserOutput{}
		out.Body = u
		return out, nil
	}, huma.OperationTags("users")); err != nil {
		panic(err)
	}

	if err = httpx.GroupDelete(api, "/users/{id}", func(ctx context.Context, input *DeleteUserInput) (*DeleteUserOutput, error) {
		deleted := store.Delete(input.ID)
		if !deleted {
			return nil, httpx.NewError(404, "user not found")
		}
		out := &DeleteUserOutput{}
		out.Body.Deleted = true
		return out, nil
	}, huma.OperationTags("users")); err != nil {
		panic(err)
	}

	fmt.Println("Fiber example server running at :8080")
	fmt.Println("GET  /health")
	fmt.Println("GET  /api/v1/users?limit=10&page=1&q=alice")
	fmt.Println("GET  /api/v1/users/{id}")
	fmt.Println("POST /api/v1/users")
	fmt.Println("PUT  /api/v1/users/{id}")
	fmt.Println("DELETE /api/v1/users/{id}")
	fmt.Println("OpenAPI: http://localhost:8080/openapi.json")
	fmt.Println("Docs:    http://localhost:8080/docs")

	if err = server.ListenAndServe(":8080"); err != nil {
		panic(err)
	}
}
