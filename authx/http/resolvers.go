package authhttp

import (
	"context"

	"github.com/DaiYuANg/archgo/authx"
)

// CredentialResolver resolves auth credential from HTTP request shape.
type CredentialResolver interface {
	ResolveCredential(ctx context.Context, req RequestInfo) (any, error)
}

// AuthorizationResolver resolves AuthorizationModel from HTTP request + principal.
type AuthorizationResolver interface {
	ResolveAuthorization(ctx context.Context, req RequestInfo, principal any) (authx.AuthorizationModel, error)
}

// CredentialResolverFunc is function adapter for CredentialResolver.
type CredentialResolverFunc func(ctx context.Context, req RequestInfo) (any, error)

func (fn CredentialResolverFunc) ResolveCredential(ctx context.Context, req RequestInfo) (any, error) {
	return fn(ctx, req)
}

func toCredentialResolverFunc(resolver CredentialResolver) CredentialResolverFunc {
	if resolver == nil {
		return nil
	}
	if fn, ok := resolver.(CredentialResolverFunc); ok {
		return fn
	}
	return func(ctx context.Context, req RequestInfo) (any, error) {
		return resolver.ResolveCredential(ctx, req)
	}
}

// AuthorizationResolverFunc is function adapter for AuthorizationResolver.
type AuthorizationResolverFunc func(
	ctx context.Context,
	req RequestInfo,
	principal any,
) (authx.AuthorizationModel, error)

func (fn AuthorizationResolverFunc) ResolveAuthorization(
	ctx context.Context,
	req RequestInfo,
	principal any,
) (authx.AuthorizationModel, error) {
	return fn(ctx, req, principal)
}

func toAuthorizationResolverFunc(resolver AuthorizationResolver) AuthorizationResolverFunc {
	if resolver == nil {
		return nil
	}
	if fn, ok := resolver.(AuthorizationResolverFunc); ok {
		return fn
	}
	return func(ctx context.Context, req RequestInfo, principal any) (authx.AuthorizationModel, error) {
		return resolver.ResolveAuthorization(ctx, req, principal)
	}
}
