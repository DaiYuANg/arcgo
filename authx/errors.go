package authx

import "errors"

var (
	// ErrInvalidAuthenticationCredential indicates that the input credential is nil or malformed.
	ErrInvalidAuthenticationCredential = errors.New("authx: invalid authentication credential")
	// ErrInvalidAuthorizationModel indicates that the authorization model is incomplete.
	ErrInvalidAuthorizationModel = errors.New("authx: invalid authorization model")
	// ErrAuthenticationProviderNotFound indicates that no provider matches the credential type.
	ErrAuthenticationProviderNotFound = errors.New("authx: authentication provider not found")
	// ErrAuthenticationManagerNotConfigured indicates that Engine has no authentication manager.
	ErrAuthenticationManagerNotConfigured = errors.New("authx: authentication manager not configured")
	// ErrAuthorizerNotConfigured indicates that Engine has no authorizer.
	ErrAuthorizerNotConfigured = errors.New("authx: authorizer not configured")
	// ErrUnauthenticated indicates that authentication failed.
	ErrUnauthenticated = errors.New("authx: unauthenticated")
)
