package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/DaiYuANg/archgo/authx"
	authhttp "github.com/DaiYuANg/archgo/authx/http"
	"github.com/DaiYuANg/archgo/authx/http/examples/shared"
	authstd "github.com/DaiYuANg/archgo/authx/http/std"
	"github.com/DaiYuANg/archgo/logx"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

type jwtCredential struct {
	Token string
}

type jwtClaims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

var demoJWTSecret = []byte("arcgo-demo-secret")

var (
	jwtActionResolver = shared.NewMethodActionResolver(map[string]string{
		http.MethodGet:    "query",
		http.MethodDelete: "delete",
	})
	jwtResourceResolver = shared.NewRouteResourceResolver(
		map[string]string{
			"/orders/{id}": "order",
		},
		map[string]string{
			"/orders/": "order",
		},
	)
)

func main() {
	logger := logx.MustNew(logx.WithConsole(true), logx.WithInfoLevel()).With("example", "authx-http-jwt")

	adminToken, err := issueDemoJWT("admin-1", []string{"user", "admin"}, demoJWTSecret, time.Now().Add(24*time.Hour))
	if err != nil {
		logger.Error("issue admin token failed", "error", err)
		os.Exit(1)
	}
	userToken, err := issueDemoJWT("user-1", []string{"user"}, demoJWTSecret, time.Now().Add(24*time.Hour))
	if err != nil {
		logger.Error("issue user token failed", "error", err)
		os.Exit(1)
	}

	router := chi.NewRouter()
	router.Use(shared.CHIRouteMetaMiddleware)
	router.Use(authstd.Require(newJWTGuard()))

	router.Get("/orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		writePrincipal(w, r)
	})
	router.Delete("/orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		writePrincipal(w, r)
	})

	logger.Info("jwt std example listening", "addr", ":8084")
	logger.Info("try query", "command", `curl -H "Authorization: Bearer `+userToken+`" http://127.0.0.1:8084/orders/1001`)
	logger.Info("try delete (forbidden)", "command", `curl -X DELETE -H "Authorization: Bearer `+userToken+`" http://127.0.0.1:8084/orders/1001`)
	logger.Info("try delete (allowed)", "command", `curl -X DELETE -H "Authorization: Bearer `+adminToken+`" http://127.0.0.1:8084/orders/1001`)

	if err = http.ListenAndServe(":8084", router); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func newJWTGuard() *authhttp.Guard {
	engine := authx.NewEngine(
		authx.WithAuthenticationManager(newJWTManager()),
		authx.WithAuthorizer(newJWTAuthorizer()),
	)

	return authhttp.NewGuard(
		engine,
		authhttp.WithCredentialResolverFunc(resolveJWTCredential),
		authhttp.WithAuthorizationResolverFunc(resolveJWTAuthorization),
	)
}

func newJWTManager() *authx.ProviderManager {
	return authx.NewProviderManager(
		authx.NewAuthenticationProviderFunc(func(_ context.Context, credential jwtCredential) (authx.AuthenticationResult, error) {
			claims := &jwtClaims{}
			token, err := jwt.ParseWithClaims(
				credential.Token,
				claims,
				func(token *jwt.Token) (any, error) {
					if token.Method != jwt.SigningMethodHS256 {
						return nil, errors.New("unsupported signing method")
					}
					return demoJWTSecret, nil
				},
			)
			if err != nil || !token.Valid {
				return authx.AuthenticationResult{}, authx.ErrUnauthenticated
			}
			if claims.Subject == "" {
				return authx.AuthenticationResult{}, authx.ErrUnauthenticated
			}

			return authx.AuthenticationResult{
				Principal: authx.Principal{
					ID:    claims.Subject,
					Roles: claims.Roles,
					Attributes: map[string]any{
						"issuer": claims.Issuer,
					},
				},
			}, nil
		}),
	)
}

func newJWTAuthorizer() authx.Authorizer {
	return authx.AuthorizerFunc(func(_ context.Context, input authx.AuthorizationModel) (authx.Decision, error) {
		if input.Resource != "order" {
			return authx.Decision{Allowed: false, Reason: "resource_not_supported"}, nil
		}

		switch input.Action {
		case "query":
			return authx.Decision{Allowed: true}, nil
		case "delete":
			principal, ok := input.Principal.(authx.Principal)
			if !ok || !shared.HasRole(principal.Roles, "admin") {
				return authx.Decision{Allowed: false, Reason: "admin_required"}, nil
			}
			return authx.Decision{Allowed: true}, nil
		default:
			return authx.Decision{Allowed: false, Reason: "action_not_supported"}, nil
		}
	})
}

func resolveJWTCredential(_ context.Context, req authhttp.RequestInfo) (any, error) {
	token, ok := shared.ParseBearer(req.Header("Authorization"))
	if !ok {
		return nil, authx.ErrInvalidAuthenticationCredential
	}
	return jwtCredential{Token: token}, nil
}

func resolveJWTAuthorization(_ context.Context, req authhttp.RequestInfo, principal any) (authx.AuthorizationModel, error) {
	action, err := jwtActionResolver.Resolve(req.Method)
	if err != nil {
		return authx.AuthorizationModel{}, err
	}
	resource, err := jwtResourceResolver.Resolve(req.RoutePattern)
	if err != nil {
		return authx.AuthorizationModel{}, err
	}

	return authx.AuthorizationModel{
		Principal: principal,
		Action:    action,
		Resource:  resource,
		Context: map[string]any{
			"order_id":      req.PathParam("id"),
			"route_pattern": req.RoutePattern,
		},
	}, nil
}

func issueDemoJWT(subject string, roles []string, secret []byte, expiresAt time.Time) (string, error) {
	claims := jwtClaims{
		Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "arcgo-authx-example",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func writePrincipal(w http.ResponseWriter, r *http.Request) {
	principal, _ := authx.PrincipalFromContextAs[authx.Principal](r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"principal_id": principal.ID,
		"roles":        principal.Roles,
		"path":         r.URL.Path,
	})
}
