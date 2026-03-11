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

	logger, err := logx.New(logx.WithConsole(true), logx.WithLevel(logx.DebugLevel))
	if err != nil {
		panic(err)
	}
	defer func() { _ = logx.Close(logger) }()

	manager, err := authx.NewManager(
		authx.WithLogger(logger),
		authx.WithProvider(identityProvider),
	)
	if err != nil {
		panic(err)
	}

	_, authentication, err := manager.AuthenticatePassword(context.Background(), "alice", "secret")
	if err != nil {
		panic(err)
	}

	fmt.Printf("authenticated=%v id=%s name=%s\n",
		authentication.IsAuthenticated(),
		authentication.Identity().ID(),
		authentication.Identity().Name(),
	)
}
