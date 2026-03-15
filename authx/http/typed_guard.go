package authhttp

import (
	"context"

	"github.com/DaiYuANg/archgo/authx"
)

type TypedCredentialResolverFunc[C any] func(ctx context.Context, req RequestInfo) (C, error)

type TypedAuthorizationResolverFunc[P any] func(
	ctx context.Context,
	req RequestInfo,
	principal P,
) (authx.AuthorizationModel, error)

type TypedOption[C any, P any] func(*TypedGuard[C, P])

// TypedGuard is an optional generic fast path that avoids request-path type assertions in resolvers.
type TypedGuard[C any, P any] struct {
	engine                *authx.Engine
	credentialResolver    TypedCredentialResolverFunc[C]
	authorizationResolver TypedAuthorizationResolverFunc[P]
}

func NewTypedGuard[C any, P any](engine *authx.Engine, opts ...TypedOption[C, P]) *TypedGuard[C, P] {
	guard := &TypedGuard[C, P]{engine: engine}
	for _, opt := range opts {
		if opt != nil {
			opt(guard)
		}
	}
	return guard
}

func WithTypedCredentialResolverFunc[C any, P any](resolver TypedCredentialResolverFunc[C]) TypedOption[C, P] {
	return func(guard *TypedGuard[C, P]) {
		guard.credentialResolver = resolver
	}
}

func WithTypedAuthorizationResolverFunc[C any, P any](
	resolver TypedAuthorizationResolverFunc[P],
) TypedOption[C, P] {
	return func(guard *TypedGuard[C, P]) {
		guard.authorizationResolver = resolver
	}
}

func (guard *TypedGuard[C, P]) Engine() *authx.Engine {
	if guard == nil {
		return nil
	}
	return guard.engine
}

func (guard *TypedGuard[C, P]) Check(
	ctx context.Context,
	req RequestInfo,
) (authx.AuthenticationResult, error) {
	if guard == nil || guard.engine == nil {
		return authx.AuthenticationResult{}, ErrNilEngine
	}
	if guard.credentialResolver == nil {
		return authx.AuthenticationResult{}, ErrCredentialResolverNotConfigured
	}

	credential, err := guard.credentialResolver(ctx, req)
	if err != nil {
		return authx.AuthenticationResult{}, err
	}

	return guard.engine.Check(ctx, credential)
}

func (guard *TypedGuard[C, P]) Can(
	ctx context.Context,
	req RequestInfo,
	principal P,
) (authx.Decision, error) {
	if guard == nil || guard.engine == nil {
		return authx.Decision{}, ErrNilEngine
	}
	if guard.authorizationResolver == nil {
		return authx.Decision{}, ErrAuthorizationResolverNotConfigured
	}

	model, err := guard.authorizationResolver(ctx, req, principal)
	if err != nil {
		return authx.Decision{}, err
	}

	return guard.engine.Can(ctx, model)
}

func (guard *TypedGuard[C, P]) Require(
	ctx context.Context,
	req RequestInfo,
) (authx.AuthenticationResult, authx.Decision, error) {
	result, principal, decision, err := guard.requireTyped(ctx, req)
	if err != nil {
		return authx.AuthenticationResult{}, authx.Decision{}, err
	}
	_ = principal
	return result, decision, nil
}

func (guard *TypedGuard[C, P]) RequireTyped(
	ctx context.Context,
	req RequestInfo,
) (P, authx.Decision, error) {
	_, principal, decision, err := guard.requireTyped(ctx, req)
	return principal, decision, err
}

func (guard *TypedGuard[C, P]) requireTyped(
	ctx context.Context,
	req RequestInfo,
) (authx.AuthenticationResult, P, authx.Decision, error) {
	var zeroPrincipal P

	if guard == nil || guard.engine == nil {
		return authx.AuthenticationResult{}, zeroPrincipal, authx.Decision{}, ErrNilEngine
	}
	if guard.credentialResolver == nil {
		return authx.AuthenticationResult{}, zeroPrincipal, authx.Decision{}, ErrCredentialResolverNotConfigured
	}
	if guard.authorizationResolver == nil {
		return authx.AuthenticationResult{}, zeroPrincipal, authx.Decision{}, ErrAuthorizationResolverNotConfigured
	}

	credential, err := guard.credentialResolver(ctx, req)
	if err != nil {
		return authx.AuthenticationResult{}, zeroPrincipal, authx.Decision{}, err
	}

	result, err := guard.engine.Check(ctx, credential)
	if err != nil {
		return authx.AuthenticationResult{}, zeroPrincipal, authx.Decision{}, err
	}
	if result.Principal == nil {
		return authx.AuthenticationResult{}, zeroPrincipal, authx.Decision{}, ErrPrincipalNotFound
	}

	principal, ok := result.Principal.(P)
	if !ok {
		return authx.AuthenticationResult{}, zeroPrincipal, authx.Decision{}, ErrPrincipalTypeMismatch
	}

	model, err := guard.authorizationResolver(ctx, req, principal)
	if err != nil {
		return authx.AuthenticationResult{}, zeroPrincipal, authx.Decision{}, err
	}

	decision, err := guard.engine.Can(ctx, model)
	if err != nil {
		return authx.AuthenticationResult{}, zeroPrincipal, authx.Decision{}, err
	}

	return result, principal, decision, nil
}

// AsGuard adapts typed guard to the classic Guard API (for middleware integrations).
func (guard *TypedGuard[C, P]) AsGuard() *Guard {
	if guard == nil {
		return nil
	}
	return NewGuard(
		guard.engine,
		WithCredentialResolverFunc(func(ctx context.Context, req RequestInfo) (any, error) {
			if guard.credentialResolver == nil {
				return nil, ErrCredentialResolverNotConfigured
			}
			return guard.credentialResolver(ctx, req)
		}),
		WithAuthorizationResolverFunc(func(ctx context.Context, req RequestInfo, principal any) (authx.AuthorizationModel, error) {
			if guard.authorizationResolver == nil {
				return authx.AuthorizationModel{}, ErrAuthorizationResolverNotConfigured
			}
			typedPrincipal, ok := principal.(P)
			if !ok {
				return authx.AuthorizationModel{}, ErrPrincipalTypeMismatch
			}
			return guard.authorizationResolver(ctx, req, typedPrincipal)
		}),
	)
}
