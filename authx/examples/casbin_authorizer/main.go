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

	identityProvider := authx.NewInMemoryIdentityProvider()
	if err := identityProvider.UpsertUser(authx.UserDetails{
		ID:           "u-1",
		Principal:    "alice",
		PasswordHash: string(hashedPassword),
		Name:         "Alice",
	}); err != nil {
		panic(err)
	}

	policySource := authx.NewMemoryPolicySource(authx.MemoryPolicySourceConfig{
		Name: "casbin-policy",
		InitialPermissions: []authx.PermissionRule{
			authx.AllowPermission("u-1", "order:1001", "read"),
		},
		InitialRoleBindings: nil,
	})

	logger, err := logx.New(logx.WithConsole(true), logx.WithLevel(logx.DebugLevel))
	if err != nil {
		panic(err)
	}
	defer func() { _ = logx.Close(logger) }()

	manager, err := authx.NewManager(
		authx.WithLogger(logger),
		authx.WithSource(policySource),
		authx.WithProvider(identityProvider),
	)
	if err != nil {
		panic(err)
	}

	_, err = manager.LoadPolicies(context.Background())
	if err != nil {
		panic(err)
	}

	ctx, _, err := manager.AuthenticatePassword(context.Background(), "alice", "secret")
	if err != nil {
		panic(err)
	}

	allowed, err := manager.Can(ctx, "read", "order:1001")
	if err != nil {
		panic(err)
	}
	fmt.Printf("before reload allowed=%v\n", allowed)

	policySource.UpdateSnapshot(authx.PolicySnapshot{
		Permissions: []authx.PermissionRule{
			authx.DenyPermission("u-1", "order:1001", "read"),
		},
		RoleBindings: nil,
	})

	version, err := manager.LoadPolicies(context.Background())
	if err != nil {
		panic(err)
	}

	allowed, err = manager.Can(ctx, "read", "order:1001")
	if err != nil {
		panic(err)
	}
	fmt.Printf("after reload allowed=%v policyVersion=%d\n", allowed, version)
}
