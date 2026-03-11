package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/DaiYuANg/arcgo/authx"
	"github.com/DaiYuANg/arcgo/logx"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type sqliteUser struct {
	id           string
	passwordHash string
	name         string
}

type sqliteMappedProvider struct {
	db *sql.DB
}

func (p sqliteMappedProvider) LoadByPrincipal(ctx context.Context, principal string) (sqliteUser, error) {
	return loadUserByPrincipal(ctx, p.db, principal)
}

func (p sqliteMappedProvider) MapToUserDetails(ctx context.Context, principal string, user sqliteUser) (authx.UserDetails, error) {
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

	db, err := sql.Open("sqlite", "file:authx_example?mode=memory&cache=shared")
	if err != nil {
		panic(err)
	}
	defer func() { _ = db.Close() }()

	if err := initUserTable(ctx, db); err != nil {
		panic(err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	if err := insertUser(ctx, db, "u-1", "alice", string(hashedPassword), "Alice"); err != nil {
		panic(err)
	}

	policySource := authx.NewMemoryPolicySource(authx.MemoryPolicySourceConfig{
		Name: "sqlite-auth-policy-source",
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
		authx.WithMappedProvider(sqliteMappedProvider{db: db}),
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

	principal, ok := authx.CurrentPrincipalAs[sqliteUser](authCtx)
	if !ok {
		panic("principal type mismatch")
	}

	fmt.Printf("sqlite auth success user=%s principalName=%s policyVersion=%d allowed=%v\n",
		authentication.Identity().ID(),
		principal.name,
		version,
		allowed,
	)
}

func initUserTable(ctx context.Context, db *sql.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	principal TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	display_name TEXT NOT NULL
)`
	_, err := db.ExecContext(ctx, ddl)
	return err
}

func insertUser(ctx context.Context, db *sql.DB, id, principal, passwordHash, displayName string) error {
	const insertSQL = `
INSERT INTO users (id, principal, password_hash, display_name)
VALUES (?, ?, ?, ?)`
	_, err := db.ExecContext(ctx, insertSQL, id, principal, passwordHash, displayName)
	return err
}

func loadUserByPrincipal(ctx context.Context, db *sql.DB, principal string) (sqliteUser, error) {
	const query = `
SELECT id, password_hash, display_name
FROM users
WHERE principal = ?`

	user := sqliteUser{}
	err := db.QueryRowContext(ctx, query, principal).Scan(&user.id, &user.passwordHash, &user.name)
	if err == sql.ErrNoRows {
		return sqliteUser{}, authx.ErrUnauthorized
	}
	if err != nil {
		return sqliteUser{}, err
	}
	return user, nil
}
