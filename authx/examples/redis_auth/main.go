package main

import (
	"context"
	"fmt"
	"os"

	"github.com/DaiYuANg/arcgo/authx"
	"github.com/DaiYuANg/arcgo/logx"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"
	"golang.org/x/crypto/bcrypt"
)

type redisUser struct {
	id           string
	passwordHash string
	name         string
}

type redisMappedProvider struct {
	client *redis.Client
}

func (p redisMappedProvider) LoadByPrincipal(ctx context.Context, principal string) (redisUser, error) {
	return loadUserByPrincipal(ctx, p.client, principal)
}

func (p redisMappedProvider) MapToUserDetails(ctx context.Context, principal string, user redisUser) (authx.UserDetails, error) {
	_ = ctx
	return authx.UserDetails{
		ID:           user.id,
		Principal:    principal,
		PasswordHash: user.passwordHash,
		Name:         user.name,
	}, nil
}

func main() {
	ctx := context.Background()

	addr := lo.Ternary(os.Getenv("REDIS_ADDR") != "", os.Getenv("REDIS_ADDR"), "127.0.0.1:6379")
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer func() { _ = client.Close() }()

	if err := client.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	if err := seedUser(ctx, client, "alice", redisUser{
		id:           "u-1",
		passwordHash: string(hashedPassword),
		name:         "Alice",
	}); err != nil {
		panic(err)
	}

	policySource := authx.NewMemoryPolicySource(authx.MemoryPolicySourceConfig{
		Name: "redis-auth-policy-source",
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
		authx.WithSource(policySource),
		authx.WithMappedProvider(redisMappedProvider{client: client}),
	)
	if err != nil {
		panic(err)
	}

	version, err := manager.LoadPolicies(ctx)
	if err != nil {
		panic(err)
	}

	authCtx, authentication, err := manager.AuthenticatePassword(ctx, "alice", "secret")
	if err != nil {
		panic(err)
	}

	allowed, err := manager.Can(authCtx, "read", "order:1001")
	if err != nil {
		panic(err)
	}

	principal, ok := authx.CurrentPrincipalAs[redisUser](authCtx)
	if !ok {
		panic("principal type mismatch")
	}

	fmt.Printf("redis auth success user=%s principalName=%s policyVersion=%d allowed=%v\n",
		authentication.Identity().ID(),
		principal.name,
		version,
		allowed,
	)
}

func seedUser(ctx context.Context, client *redis.Client, principal string, user redisUser) error {
	key := userKey(principal)
	values := map[string]any{
		"id":            user.id,
		"password_hash": user.passwordHash,
		"display_name":  user.name,
	}
	return client.HSet(ctx, key, values).Err()
}

func loadUserByPrincipal(ctx context.Context, client *redis.Client, principal string) (redisUser, error) {
	data, err := client.HGetAll(ctx, userKey(principal)).Result()
	if err != nil {
		return redisUser{}, err
	}
	if len(data) == 0 {
		return redisUser{}, authx.ErrUnauthorized
	}

	return redisUser{
		id:           data["id"],
		passwordHash: data["password_hash"],
		name:         data["display_name"],
	}, nil
}

func userKey(principal string) string {
	return "authx:user:" + principal
}
