package authhttp

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/authx"
)

// Option configures Guard behavior.
type Option func(*Guard)

// Guard drives authx Check/Can flow for HTTP integrations.
type Guard struct {
	engine                *authx.Engine
	credentialResolver    CredentialResolverFunc
	authorizationResolver AuthorizationResolverFunc
}

// NewGuard constructs a Guard from engine and opts.
func NewGuard(engine *authx.Engine, opts ...Option) *Guard {
	guard := &Guard{engine: engine}
	ApplyOptions(guard, opts...)
	return guard
}

// WithCredentialResolver configures how Guard reads credentials from a request.
func WithCredentialResolver(resolver CredentialResolver) Option {
	return func(guard *Guard) {
		guard.credentialResolver = toCredentialResolverFunc(resolver)
	}
}

// WithAuthorizationResolver configures how Guard builds the authorization model.
func WithAuthorizationResolver(resolver AuthorizationResolver) Option {
	return func(guard *Guard) {
		guard.authorizationResolver = toAuthorizationResolverFunc(resolver)
	}
}

// WithCredentialResolverFunc configures Guard with a function-based credential resolver.
func WithCredentialResolverFunc(resolver CredentialResolverFunc) Option {
	return func(guard *Guard) {
		guard.credentialResolver = resolver
	}
}

// WithAuthorizationResolverFunc configures Guard with a function-based authorization resolver.
func WithAuthorizationResolverFunc(resolver AuthorizationResolverFunc) Option {
	return func(guard *Guard) {
		guard.authorizationResolver = resolver
	}
}

// Engine returns the underlying authx engine.
func (guard *Guard) Engine() *authx.Engine {
	if guard == nil {
		return nil
	}
	return guard.engine
}

// Check runs engine.Check with credential resolved from request.
func (guard *Guard) Check(
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

	result, err := guard.engine.Check(ctx, credential)
	if err != nil {
		return authx.AuthenticationResult{}, fmt.Errorf("check request credential: %w", err)
	}
	return result, nil
}

// Can runs engine.Can from resolved AuthorizationModel.
func (guard *Guard) Can(
	ctx context.Context,
	req RequestInfo,
	principal any,
) (authx.Decision, error) {
	if guard == nil || guard.engine == nil {
		return authx.Decision{}, ErrNilEngine
	}
	if guard.authorizationResolver == nil {
		return authx.Decision{}, ErrAuthorizationResolverNotConfigured
	}
	if principal == nil {
		return authx.Decision{}, ErrPrincipalNotFound
	}

	model, err := guard.authorizationResolver(ctx, req, principal)
	if err != nil {
		return authx.Decision{}, err
	}

	decision, err := guard.engine.Can(ctx, model)
	if err != nil {
		return authx.Decision{}, fmt.Errorf("authorize request: %w", err)
	}
	return decision, nil
}

// Require runs Check then Can and returns both result/decision.
func (guard *Guard) Require(
	ctx context.Context,
	req RequestInfo,
) (authx.AuthenticationResult, authx.Decision, error) {
	if guard == nil || guard.engine == nil {
		return authx.AuthenticationResult{}, authx.Decision{}, ErrNilEngine
	}
	if guard.credentialResolver == nil {
		return authx.AuthenticationResult{}, authx.Decision{}, ErrCredentialResolverNotConfigured
	}
	if guard.authorizationResolver == nil {
		return authx.AuthenticationResult{}, authx.Decision{}, ErrAuthorizationResolverNotConfigured
	}

	credential, err := guard.credentialResolver(ctx, req)
	if err != nil {
		return authx.AuthenticationResult{}, authx.Decision{}, err
	}

	result, err := guard.engine.Check(ctx, credential)
	if err != nil {
		return authx.AuthenticationResult{}, authx.Decision{}, fmt.Errorf("check request credential: %w", err)
	}

	if result.Principal == nil {
		return authx.AuthenticationResult{}, authx.Decision{}, ErrPrincipalNotFound
	}

	model, err := guard.authorizationResolver(ctx, req, result.Principal)
	if err != nil {
		return authx.AuthenticationResult{}, authx.Decision{}, err
	}

	decision, err := guard.engine.Can(ctx, model)
	if err != nil {
		return authx.AuthenticationResult{}, authx.Decision{}, fmt.Errorf("authorize request: %w", err)
	}

	return result, decision, nil
}
