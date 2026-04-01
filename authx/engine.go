package authx

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sync"

	"github.com/DaiYuANg/arcgo/pkg/option"
	"github.com/samber/lo"
)

// Engine separates authentication (Check) and authorization (Can).
type Engine struct {
	mu     sync.RWMutex
	authn  AuthenticationManager
	authz  Authorizer
	hooks  []Hook
	logger *slog.Logger
	debug  bool
}

// NewEngine constructs an Engine from opts.
func NewEngine(opts ...EngineOption) *Engine {
	engine := &Engine{logger: slog.Default()}
	option.Apply(engine, opts...)
	engine.logDebug("authx engine created", "hooks", len(engine.hooks), "has_authn", engine.authn != nil, "has_authz", engine.authz != nil)
	return engine
}

// SetAuthenticationManager updates the authentication manager used by Check.
func (engine *Engine) SetAuthenticationManager(manager AuthenticationManager) {
	if engine == nil {
		return
	}
	engine.mu.Lock()
	engine.authn = manager
	engine.mu.Unlock()
	engine.logDebug("authentication manager configured", "manager_type", reflect.TypeOf(manager))
}

// SetAuthorizer updates the authorizer used by Can.
func (engine *Engine) SetAuthorizer(authorizer Authorizer) {
	if engine == nil {
		return
	}
	engine.mu.Lock()
	engine.authz = authorizer
	engine.mu.Unlock()
	engine.logDebug("authorizer configured", "authorizer_type", reflect.TypeOf(authorizer))
}

// AddHook appends hook to the engine lifecycle hooks.
func (engine *Engine) AddHook(hook Hook) {
	if engine == nil || hook == nil {
		return
	}
	engine.mu.Lock()
	engine.hooks = lo.Concat(engine.hooks, []Hook{hook})
	engine.mu.Unlock()
	engine.logDebug("authx hook added", "hook_type", reflect.TypeOf(hook), "hooks", len(engine.hooks))
}

// Check authenticates credential and returns principal.
func (engine *Engine) Check(ctx context.Context, credential any) (AuthenticationResult, error) {
	if credential == nil {
		return AuthenticationResult{}, ErrInvalidAuthenticationCredential
	}
	engine.logDebug("authx check started", "credential_type", reflect.TypeOf(credential))

	authn, hooks := engine.snapshotCheckDependencies()
	if authn == nil {
		engine.logError("authx check failed", "credential_type", reflect.TypeOf(credential), "error", ErrAuthenticationManagerNotConfigured)
		return AuthenticationResult{}, ErrAuthenticationManagerNotConfigured
	}

	for _, hook := range hooks {
		if err := hook.BeforeCheck(ctx, credential); err != nil {
			engine.logError("authx check before hook failed", "credential_type", reflect.TypeOf(credential), "error", err)
			return AuthenticationResult{}, fmt.Errorf("before check hook: %w", err)
		}
	}

	result, err := authn.Authenticate(ctx, credential)
	lo.ForEach(hooks, func(hook Hook, _ int) {
		hook.AfterCheck(ctx, credential, result, err)
	})
	if err != nil {
		engine.logError("authx check failed", "credential_type", reflect.TypeOf(credential), "error", err)
		return AuthenticationResult{}, fmt.Errorf("authenticate credential: %w", err)
	}
	engine.logDebug("authx check completed", "credential_type", reflect.TypeOf(credential), "principal_type", reflect.TypeOf(result.Principal))
	return result, nil
}

// Can authorizes principal access to action/resource.
func (engine *Engine) Can(ctx context.Context, input AuthorizationModel) (Decision, error) {
	if err := validateAuthorizationModel(input); err != nil {
		return Decision{}, err
	}
	engine.logDebug("authx can started", "action", input.Action, "resource", input.Resource)

	authorizer, hooks := engine.snapshotCanDependencies()
	if authorizer == nil {
		engine.logError("authx can failed", "action", input.Action, "resource", input.Resource, "error", ErrAuthorizerNotConfigured)
		return Decision{}, ErrAuthorizerNotConfigured
	}

	for _, hook := range hooks {
		if err := hook.BeforeCan(ctx, input); err != nil {
			engine.logError("authx can before hook failed", "action", input.Action, "resource", input.Resource, "error", err)
			return Decision{}, fmt.Errorf("before authorization hook: %w", err)
		}
	}

	decision, err := authorizer.Authorize(ctx, input)
	lo.ForEach(hooks, func(hook Hook, _ int) {
		hook.AfterCan(ctx, input, decision, err)
	})
	if err != nil {
		engine.logError("authx can failed", "action", input.Action, "resource", input.Resource, "error", err)
		return Decision{}, fmt.Errorf("authorize request: %w", err)
	}
	engine.logDebug("authx can completed", "action", input.Action, "resource", input.Resource, "allowed", decision.Allowed, "policy_id", decision.PolicyID)
	return decision, nil
}

func (engine *Engine) snapshotCheckDependencies() (AuthenticationManager, []Hook) {
	if engine == nil {
		return nil, nil
	}

	engine.mu.RLock()
	authn := engine.authn
	hooks := engine.hooks
	engine.mu.RUnlock()
	return authn, hooks
}

func (engine *Engine) snapshotCanDependencies() (Authorizer, []Hook) {
	if engine == nil {
		return nil, nil
	}

	engine.mu.RLock()
	authorizer := engine.authz
	hooks := engine.hooks
	engine.mu.RUnlock()
	return authorizer, hooks
}

func validateAuthorizationModel(input AuthorizationModel) error {
	if input.Action == "" || input.Resource == "" {
		return ErrInvalidAuthorizationModel
	}
	if input.Principal == nil {
		return ErrInvalidAuthorizationModel
	}
	return nil
}

func (engine *Engine) logDebug(msg string, attrs ...any) {
	if engine == nil || engine.logger == nil || !engine.debug {
		return
	}
	engine.logger.Debug(msg, attrs...)
}

func (engine *Engine) logError(msg string, attrs ...any) {
	if engine == nil || engine.logger == nil {
		return
	}
	engine.logger.Error(msg, attrs...)
}
