package main

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/authx"
	"github.com/DaiYuANg/arcgo/logx"
	"golang.org/x/crypto/bcrypt"
)

type userRecord struct {
	ID           string
	Principal    string
	PasswordHash string
	Name         string
}

type UserRepository interface {
	FindByPrincipal(ctx context.Context, principal string) (userRecord, error)
}

type mapUserRepository struct {
	users map[string]userRecord
}

func (r *mapUserRepository) FindByPrincipal(ctx context.Context, principal string) (userRecord, error) {
	_ = ctx
	user, ok := r.users[principal]
	if !ok {
		return userRecord{}, authx.ErrUnauthorized
	}
	return user, nil
}

type repositoryIdentityProvider struct {
	repo UserRepository
}

func newRepositoryIdentityProvider(repo UserRepository) (*repositoryIdentityProvider, error) {
	if repo == nil {
		return nil, fmt.Errorf("%w: repository is nil", authx.ErrInvalidAuthenticator)
	}
	return &repositoryIdentityProvider{repo: repo}, nil
}

func (p *repositoryIdentityProvider) LoadByPrincipal(ctx context.Context, principal string) (authx.UserDetails, error) {
	record, err := p.repo.FindByPrincipal(ctx, principal)
	if err != nil {
		return authx.UserDetails{}, err
	}
	return authx.UserDetails{
		ID:           record.ID,
		Principal:    record.Principal,
		PasswordHash: record.PasswordHash,
		Name:         record.Name,
		Payload:      record,
	}, nil
}

func main() {
	ctx := context.Background()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	repo := &mapUserRepository{
		users: map[string]userRecord{
			"alice": {
				ID:           "u-1",
				Principal:    "alice",
				PasswordHash: string(hashedPassword),
				Name:         "Alice",
			},
		},
	}

	provider, err := newRepositoryIdentityProvider(repo)
	if err != nil {
		panic(err)
	}

	policySource := authx.NewMemoryPolicySource(authx.MemoryPolicySourceConfig{
		Name: "custom-provider-policy-source",
		InitialPermissions: []authx.PermissionRule{
			authx.AllowPermission("u-1", "order:1001", "read"),
		},
	})

	logger, err := logx.New(logx.WithConsole(true), logx.WithLevel(logx.DebugLevel))
	if err != nil {
		panic(err)
	}
	defer func() { _ = logx.Close(logger) }()

	manager, err := authx.NewManager(
		authx.WithLogger(logger),
		authx.WithProvider(provider),
		authx.WithSource(policySource),
	)
	if err != nil {
		panic(err)
	}

	version, err := manager.LoadPolicies(ctx)
	if err != nil {
		panic(err)
	}

	authCtx, auth, err := manager.AuthenticatePassword(ctx, "alice", "secret")
	if err != nil {
		panic(err)
	}

	allowed, err := manager.Can(authCtx, "read", "order:1001")
	if err != nil {
		panic(err)
	}

	principal, ok := authx.CurrentPrincipalAs[userRecord](authCtx)
	if !ok {
		panic("principal type mismatch")
	}

	fmt.Printf("custom provider auth user=%s principalName=%s policyVersion=%d allowed=%v\n",
		auth.Identity().ID(),
		principal.Name,
		version,
		allowed,
	)
}
