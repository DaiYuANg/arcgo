package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/examples/dbx/internal/shared"
)

type Preferences struct {
	Theme string   `json:"theme"`
	Flags []string `json:"flags"`
}

type AccountStatus string

const (
	AccountStatusActive  AccountStatus = "active"
	AccountStatusBlocked AccountStatus = "blocked"
)

func (s AccountStatus) MarshalText() ([]byte, error) {
	switch s {
	case AccountStatusActive, AccountStatusBlocked:
		return []byte(s), nil
	default:
		return nil, fmt.Errorf("invalid account status %q", s)
	}
}

func (s *AccountStatus) UnmarshalText(text []byte) error {
	value := AccountStatus(strings.ToLower(strings.TrimSpace(string(text))))
	switch value {
	case AccountStatusActive, AccountStatusBlocked:
		*s = value
		return nil
	default:
		return fmt.Errorf("invalid account status %q", value)
	}
}

type Account struct {
	ID          int64         `dbx:"id"`
	Username    string        `dbx:"username"`
	Status      AccountStatus `dbx:"status,codec=text"`
	CreatedAt   time.Time     `dbx:"created_at,codec=unix_milli_time"`
	Preferences Preferences   `dbx:"preferences,codec=json"`
	Tags        []string      `dbx:"tags,codec=csv"`
}

type AccountSchema struct {
	dbx.Schema[Account]
	ID          dbx.Column[Account, int64]         `dbx:"id,pk,auto"`
	Username    dbx.Column[Account, string]        `dbx:"username,unique"`
	Status      dbx.Column[Account, AccountStatus] `dbx:"status,type=text"`
	CreatedAt   dbx.Column[Account, time.Time]     `dbx:"created_at,type=integer"`
	Preferences dbx.Column[Account, Preferences]   `dbx:"preferences,type=text"`
	Tags        dbx.Column[Account, []string]      `dbx:"tags,type=text"`
}

func main() {
	ctx := context.Background()
	logger := shared.NewLogger()
	core, closeDB, err := shared.OpenSQLite(
		"dbx-codec",
		dbx.WithLogger(logger),
		dbx.WithDebug(true),
	)
	if err != nil {
		panic(err)
	}
	defer func() { _ = closeDB() }()

	accounts := dbx.MustSchema("accounts", AccountSchema{})
	if _, err := core.AutoMigrate(ctx, accounts); err != nil {
		panic(err)
	}

	csvCodec := dbx.NewCodec[[]string](
		"csv",
		func(src any) ([]string, error) {
			switch value := src.(type) {
			case string:
				return splitCSV(value), nil
			case []byte:
				return splitCSV(string(value)), nil
			default:
				return nil, fmt.Errorf("csv codec only supports string or []byte, got %T", src)
			}
		},
		func(values []string) (any, error) {
			return strings.Join(values, ","), nil
		},
	)
	mapper := dbx.MustMapperWithOptions[Account](accounts, dbx.WithMapperCodecs(csvCodec))
	for _, account := range []Account{
		{
			Username:  "alice",
			Status:    AccountStatusActive,
			CreatedAt: time.UnixMilli(1711111111222).UTC(),
			Preferences: Preferences{
				Theme: "dark",
				Flags: []string{"beta", "admin"},
			},
			Tags: []string{"go", "dbx", "codec"},
		},
		{
			Username:  "bob",
			Status:    AccountStatusBlocked,
			CreatedAt: time.UnixMilli(1712222222333).UTC(),
			Preferences: Preferences{
				Theme: "light",
				Flags: []string{"reader"},
			},
			Tags: []string{"sqlite", "json"},
		},
	} {
		assignments, err := mapper.InsertAssignments(core, accounts, &account)
		if err != nil {
			panic(err)
		}
		if _, err := dbx.Exec(ctx, core, dbx.InsertInto(accounts).Values(assignments...)); err != nil {
			panic(err)
		}
	}

	items, err := dbx.QueryAll[Account](
		ctx,
		core,
		dbx.Select(accounts.AllColumns()...).From(accounts).OrderBy(accounts.ID.Asc()),
		mapper,
	)
	if err != nil {
		panic(err)
	}

	fmt.Println("codec example:")
	for _, item := range items {
		fmt.Printf("- id=%d username=%s status=%s created_at=%s theme=%s tags=%v\n", item.ID, item.Username, item.Status, item.CreatedAt.Format(time.RFC3339), item.Preferences.Theme, item.Tags)
	}
}

func splitCSV(input string) []string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}
