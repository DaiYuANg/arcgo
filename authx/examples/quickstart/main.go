package main

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/authx"
	"github.com/DaiYuANg/arcgo/logx"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	primaryProvider := authx.NewInMemoryIdentityProvider()
	secondaryProvider := authx.NewInMemoryIdentityProvider()
	if err := secondaryProvider.UpsertUser(authx.UserDetails{
		ID:           "u-1",
		Principal:    "alice",
		PasswordHash: string(hashedPassword),
		Name:         "Alice",
	}); err != nil {
		panic(err)
	}

	policySource := authx.NewMemoryPolicySource(authx.MemoryPolicySourceConfig{
		Name: "quickstart-policy",
		InitialPermissions: []authx.PermissionRule{
			authx.AllowPermission("u-1", "order:1001", "read"),
			authx.AllowPermission("role:admin", "order:1001", "write"),
		},
		InitialRoleBindings: []authx.RoleBinding{
			authx.NewRoleBinding("u-1", "role:admin"),
		},
	})

	logger, err := logx.New(logx.WithConsole(true), logx.WithLevel(logx.DebugLevel))
	if err != nil {
		panic(err)
	}
	defer func() { _ = logx.Close(logger) }()

	manager, err := authx.NewManager(
		authx.WithLogger(logger),
		authx.WithSource(policySource),
		authx.WithProvider(primaryProvider),
		authx.WithProvider(secondaryProvider),
	)
	if err != nil {
		panic(err)
	}

	version, err := manager.LoadPolicies(context.Background())
	if err != nil {
		panic(err)
	}

	ctx, authentication, err := manager.AuthenticatePassword(context.Background(), "alice", "secret")
	if err != nil {
		panic(err)
	}

	allowed, err := manager.Can(ctx, "write", "order:1001")
	if err != nil {
		panic(err)
	}

	fmt.Printf("authenticated=%v user=%s policyVersion=%d allowed=%v\n",
		authentication.IsAuthenticated(),
		authentication.Identity().ID(),
		version,
		allowed,
	)
}
