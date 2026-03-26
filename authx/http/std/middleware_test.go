package std_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DaiYuANg/arcgo/authx"
	authhttp "github.com/DaiYuANg/arcgo/authx/http"
	authstd "github.com/DaiYuANg/arcgo/authx/http/std"
)

type middlewareCredential struct {
	Token string
}

func newMiddlewareGuard() *authhttp.Guard {
	manager := authx.NewProviderManager(
		authx.NewAuthenticationProviderFunc(func(_ context.Context, credential middlewareCredential) (authx.AuthenticationResult, error) {
			if credential.Token == "" {
				return authx.AuthenticationResult{}, errors.New("missing token")
			}
			return authx.AuthenticationResult{
				Principal: authx.Principal{ID: credential.Token},
			}, nil
		}),
	)

	authorizer := authx.AuthorizerFunc(func(_ context.Context, input authx.AuthorizationModel) (authx.Decision, error) {
		if input.Action == "delete" {
			return authx.Decision{Allowed: false, Reason: "no_permission"}, nil
		}
		return authx.Decision{Allowed: true}, nil
	})

	engine := authx.NewEngine(
		authx.WithAuthenticationManager(manager),
		authx.WithAuthorizer(authorizer),
	)

	return authhttp.NewGuard(
		engine,
		authhttp.WithCredentialResolverFunc(func(_ context.Context, req authhttp.RequestInfo) (any, error) {
			return middlewareCredential{Token: req.Header("Authorization")}, nil
		}),
		authhttp.WithAuthorizationResolverFunc(func(_ context.Context, req authhttp.RequestInfo, principal any) (authx.AuthorizationModel, error) {
			action := "query"
			if req.Method == http.MethodDelete {
				action = "delete"
			}
			return authx.AuthorizationModel{
				Principal: principal,
				Action:    action,
				Resource:  "order",
				Context: map[string]any{
					"order_id": req.PathParam("id"),
				},
			}, nil
		}),
	)
}

func TestRequireAllowed(t *testing.T) {
	guard := newMiddlewareGuard()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := authx.PrincipalFromContext(r.Context()); !ok {
			t.Fatalf("principal missing in request context")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	handler := authstd.Require(guard)(next)
	req := httptest.NewRequest(http.MethodGet, "/orders/123", http.NoBody)
	req.Header.Set("Authorization", "user-1")
	req = req.WithContext(authhttp.WithPathParams(authhttp.WithRoutePattern(req.Context(), "/orders/:id"), map[string]string{"id": "123"}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestRequireDenied(t *testing.T) {
	guard := newMiddlewareGuard()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	handler := authstd.Require(guard)(next)
	req := httptest.NewRequest(http.MethodDelete, "/orders/123", http.NoBody)
	req.Header.Set("Authorization", "user-1")
	req = req.WithContext(authhttp.WithPathParams(authhttp.WithRoutePattern(req.Context(), "/orders/:id"), map[string]string{"id": "123"}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}
