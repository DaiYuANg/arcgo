package authx

import (
	"context"
	"fmt"
	"reflect"
)

// TypedAuthenticationProvider keeps credential strongly typed while exposing a non-generic provider surface.
type TypedAuthenticationProvider[C any] interface {
	Authenticate(ctx context.Context, credential C) (AuthenticationResult, error)
}

// TypedAuthenticationProviderFunc is a lightweight typed provider helper.
type TypedAuthenticationProviderFunc[C any] func(ctx context.Context, credential C) (AuthenticationResult, error)

// Authenticate calls fn or returns ErrUnauthenticated when fn is nil.
func (fn TypedAuthenticationProviderFunc[C]) Authenticate(
	ctx context.Context,
	credential C,
) (AuthenticationResult, error) {
	if fn == nil {
		return AuthenticationResult{}, ErrUnauthenticated
	}
	return fn(ctx, credential)
}

// NewAuthenticationProvider wraps a typed provider into a manager-compatible provider.
func NewAuthenticationProvider[C any](provider TypedAuthenticationProvider[C]) AuthenticationProvider {
	return &typedProviderAdapter[C]{
		provider:       provider,
		credentialType: reflect.TypeFor[C](),
	}
}

// NewAuthenticationProviderFunc wraps a typed function into a manager-compatible provider.
func NewAuthenticationProviderFunc[C any](
	fn func(ctx context.Context, credential C) (AuthenticationResult, error),
) AuthenticationProvider {
	return NewAuthenticationProvider[C](TypedAuthenticationProviderFunc[C](fn))
}

type typedProviderAdapter[C any] struct {
	provider       TypedAuthenticationProvider[C]
	credentialType reflect.Type
}

func (adapter *typedProviderAdapter[C]) CredentialType() reflect.Type {
	return adapter.credentialType
}

func (adapter *typedProviderAdapter[C]) AuthenticateAny(
	ctx context.Context,
	credential any,
) (AuthenticationResult, error) {
	typedCredential, ok := credential.(C)
	if !ok {
		return AuthenticationResult{}, ErrInvalidAuthenticationCredential
	}
	result, err := adapter.provider.Authenticate(ctx, typedCredential)
	if err != nil {
		return AuthenticationResult{}, fmt.Errorf("authenticate credential: %w", err)
	}
	return result, nil
}

// AuthenticationManagerFunc is a lightweight manager helper.
type AuthenticationManagerFunc func(ctx context.Context, credential any) (AuthenticationResult, error)

// Authenticate calls fn or returns ErrAuthenticationManagerNotConfigured when fn is nil.
func (fn AuthenticationManagerFunc) Authenticate(
	ctx context.Context,
	credential any,
) (AuthenticationResult, error) {
	if fn == nil {
		return AuthenticationResult{}, ErrAuthenticationManagerNotConfigured
	}
	return fn(ctx, credential)
}

// AuthorizerFunc is a lightweight authorizer helper.
type AuthorizerFunc func(ctx context.Context, input AuthorizationModel) (Decision, error)

// Authorize calls fn or returns ErrAuthorizerNotConfigured when fn is nil.
func (fn AuthorizerFunc) Authorize(ctx context.Context, input AuthorizationModel) (Decision, error) {
	if fn == nil {
		return Decision{}, ErrAuthorizerNotConfigured
	}
	return fn(ctx, input)
}
