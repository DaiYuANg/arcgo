package authn

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/DaiYuANg/arcgo/authx"
	authhttp "github.com/DaiYuANg/arcgo/authx/http"
	authfiber "github.com/DaiYuANg/arcgo/authx/http/fiber"
	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
	collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	collectionset "github.com/DaiYuANg/arcgo/collectionx/set"
	"github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/config"
	"github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/entity"
	authsvc "github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/service/auth"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/mo"
)

type bearerCredential struct {
	Token string
}

var actionMapping = collectionmapping.NewMapFrom(map[string]string{
	http.MethodGet:    "query",
	http.MethodPost:   "create",
	http.MethodDelete: "delete",
	http.MethodPut:    "update",
	http.MethodPatch:  "update",
})

func NewAuthxEngineOptions(
	authz *authsvc.AuthorizationService,
	jwtSvc *authsvc.JWTService,
	obs observabilityx.Observability,
) []authx.EngineOption {
	return []authx.EngineOption{
		authx.WithAuthenticationManager(
			authx.NewProviderManager(
				authx.NewAuthenticationProviderFunc(func(
					ctx context.Context,
					credential bearerCredential,
				) (authx.AuthenticationResult, error) {
					ctx, span := obs.StartSpan(ctx, "rbac.auth.check")
					defer span.End()

					principal, err := jwtSvc.ParseToken(credential.Token)
					if err != nil {
						span.RecordError(err)
						obs.AddCounter(ctx, "rbac_auth_check_total", 1,
							observabilityx.String("result", "denied"),
						)
						return authx.AuthenticationResult{}, err
					}

					obs.AddCounter(ctx, "rbac_auth_check_total", 1,
						observabilityx.String("result", "ok"),
					)
					return authx.AuthenticationResult{Principal: principal}, nil
				}),
			),
		),
		authx.WithAuthorizer(authx.AuthorizerFunc(func(
			ctx context.Context,
			input authx.AuthorizationModel,
		) (authx.Decision, error) {
			ctx, span := obs.StartSpan(ctx, "rbac.auth.can")
			defer span.End()

			principal, ok := input.Principal.(entity.Principal)
			if !ok {
				return authx.Decision{Allowed: false, Reason: "invalid_principal"}, nil
			}
			allowed, err := authz.Can(ctx, principal.UserID, input.Action, input.Resource)
			if err != nil {
				span.RecordError(err)
				return authx.Decision{}, err
			}
			if !allowed {
				obs.AddCounter(ctx, "rbac_auth_can_total", 1,
					observabilityx.String("result", "denied"),
					observabilityx.String("action", input.Action),
					observabilityx.String("resource", input.Resource),
				)
				return authx.Decision{Allowed: false, Reason: "permission_denied"}, nil
			}
			obs.AddCounter(ctx, "rbac_auth_can_total", 1,
				observabilityx.String("result", "ok"),
				observabilityx.String("action", input.Action),
				observabilityx.String("resource", input.Resource),
			)
			return authx.Decision{Allowed: true}, nil
		})),
	}
}

func NewGuard(engine *authx.Engine) *authhttp.Guard {
	return authhttp.NewGuard(
		engine,
		authhttp.WithCredentialResolverFunc(resolveCredential),
		authhttp.WithAuthorizationResolverFunc(resolveAuthorization),
	)
}

func NewAuthMiddleware(cfg config.AppConfig, guard *authhttp.Guard) fiber.Handler {
	require := authfiber.RequireFast(guard)
	loginPath := strings.TrimRight(cfg.BasePath(), "/") + "/login"
	docsPrefix := strings.TrimRight(cfg.DocsPath(), "/") + "/"

	publicPaths := collectionset.NewSet(
		"/health",
		cfg.MetricsPath(),
		cfg.DocsPath(),
		cfg.OpenAPIPath(),
		loginPath,
	)
	publicPrefixes := collectionlist.NewList(docsPrefix, "/schemas/")

	return func(c *fiber.Ctx) error {
		path := c.Path()
		if publicPaths.Contains(path) {
			return c.Next()
		}
		isPublicPrefix := false
		publicPrefixes.Range(func(_ int, prefix string) bool {
			if strings.HasPrefix(path, prefix) {
				isPublicPrefix = true
				return false
			}
			return true
		})
		if isPublicPrefix {
			return c.Next()
		}
		return require(c)
	}
}

func resolveCredential(_ context.Context, req authhttp.RequestInfo) (any, error) {
	raw := strings.TrimSpace(req.Header("Authorization"))
	parts := strings.Fields(raw)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, fmt.Errorf("%w: missing bearer token", authx.ErrInvalidAuthenticationCredential)
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return nil, fmt.Errorf("%w: empty bearer token", authx.ErrInvalidAuthenticationCredential)
	}
	return bearerCredential{Token: token}, nil
}

func resolveAuthorization(
	_ context.Context,
	req authhttp.RequestInfo,
	principal any,
) (authx.AuthorizationModel, error) {
	action, err := resolveAction(req.Method)
	if err != nil {
		return authx.AuthorizationModel{}, err
	}
	resource, err := resolveResource(req)
	if err != nil {
		return authx.AuthorizationModel{}, err
	}
	return authx.AuthorizationModel{
		Principal: principal,
		Action:    action,
		Resource:  resource,
		Context: map[string]any{
			"route_pattern": req.RoutePattern,
			"path":          req.Path,
		},
	}, nil
}

func resolveAction(method string) (string, error) {
	actionOpt := mo.TupleToOption(actionMapping.Get(strings.ToUpper(strings.TrimSpace(method))))
	action, ok := actionOpt.Get()
	if !ok {
		return "", fmt.Errorf("unsupported method for action mapping: %s", method)
	}
	return action, nil
}

func resolveResource(req authhttp.RequestInfo) (string, error) {
	pattern := strings.TrimSpace(req.RoutePattern)
	if pattern == "" {
		pattern = strings.TrimSpace(req.Path)
	}
	if strings.Contains(pattern, "/books") {
		return "book", nil
	}
	if strings.Contains(pattern, "/users") {
		return "user", nil
	}
	if strings.Contains(pattern, "/roles") {
		return "role", nil
	}
	return "", fmt.Errorf("unsupported route pattern for resource mapping: %s", pattern)
}
