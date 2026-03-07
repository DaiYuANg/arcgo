package httpx

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/DaiYuANg/arcgo/collectionx/set"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
)

// Server is the central httpx runtime object used to register routes and expose
// Huma/OpenAPI capabilities.
type Server struct {
	adapter            adapter.Adapter
	basePath           string
	routes             *list.ConcurrentList[RouteInfo]
	routeKeys          *set.ConcurrentSet[string]
	logger             *slog.Logger
	printRoutes        bool
	validator          *validator.Validate
	panicRecover       bool
	accessLog          bool
	humaOptions        adapter.HumaOptions
	openAPIPatches     []func(*huma.OpenAPI)
	humaMiddlewares    []func(huma.Context, func(huma.Context))
	operationModifiers []func(*huma.Operation)
}

// ServerOption mutates a server during construction.
type ServerOption func(*Server)

// NewServer constructs a server, creating a default std adapter when none is provided.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		logger:       slog.Default(),
		routes:       list.NewConcurrentList[RouteInfo](),
		routeKeys:    set.NewConcurrentSet[string](),
		panicRecover: true,
	}

	lo.ForEach(opts, func(opt ServerOption, _ int) {
		opt(s)
	})

	if s.adapter == nil {
		s.adapter = std.New(s.humaOptions)
	}

	s.applyPendingHumaConfig()

	return s
}
