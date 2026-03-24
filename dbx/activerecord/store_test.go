package activerecord

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DaiYuANg/arcgo/dbx"
	sqlitedialect "github.com/DaiYuANg/arcgo/dbx/dialect/sqlite"
	"github.com/DaiYuANg/arcgo/dbx/repository"
	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID   int64  `dbx:"id"`
	Name string `dbx:"name"`
}

type UserSchema struct {
	dbx.Schema[User]
	ID   dbx.IDColumn[User, int64, dbx.IDSnowflake] `dbx:"id,pk"`
	Name dbx.Column[User, string] `dbx:"name"`
}

func TestModelSaveReloadDelete(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite3", "file:activerecord_model_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer raw.Close()
	core := dbx.MustNewWithOptions(raw, sqlitedialect.New())
	users := dbx.MustSchema("users", UserSchema{})
	if _, err := core.AutoMigrate(ctx, users); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	store := New[User](core, users)
	model := store.Wrap(&User{Name: "alice"})
	if err := model.Save(ctx); err != nil {
		t.Fatalf("save create: %v", err)
	}
	if model.Entity().ID == 0 {
		t.Fatal("expected id generated after save")
	}

	model.Entity().Name = "alice-v2"
	if err := model.Save(ctx); err != nil {
		t.Fatalf("save update: %v", err)
	}

	found, err := store.FindByID(ctx, model.Entity().ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if found.Entity().Name != "alice-v2" {
		t.Fatalf("unexpected found entity: %+v", found.Entity())
	}

	model.Entity().Name = "stale"
	if err := model.Reload(ctx); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if model.Entity().Name != "alice-v2" {
		t.Fatalf("reload did not refresh entity: %+v", model.Entity())
	}

	if err := model.Delete(ctx); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = store.FindByID(ctx, model.Entity().ID)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected repository.ErrNotFound, got: %v", err)
	}
}

func TestStoreFindOptionAPIs(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite3", "file:activerecord_option_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer raw.Close()
	core := dbx.MustNewWithOptions(raw, sqlitedialect.New())
	users := dbx.MustSchema("users", UserSchema{})
	if _, err := core.AutoMigrate(ctx, users); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	store := New[User](core, users)
	model := store.Wrap(&User{Name: "alice"})
	if err := model.Save(ctx); err != nil {
		t.Fatalf("seed: %v", err)
	}

	noneByID, err := store.FindByIDOption(ctx, int64(99999))
	if err != nil {
		t.Fatalf("find by id option: %v", err)
	}
	if noneByID.IsPresent() {
		t.Fatal("expected absent option for missing id")
	}

	byID, err := store.FindByIDOption(ctx, model.Entity().ID)
	if err != nil {
		t.Fatalf("find by id option existing: %v", err)
	}
	found, ok := byID.Get()
	if !ok || found.Entity().Name != "alice" {
		t.Fatalf("expected alice from option, got ok=%v model=%+v", ok, found)
	}

	key := found.Key()
	byKey, err := store.FindByKeyOption(ctx, key)
	if err != nil {
		t.Fatalf("find by key option existing: %v", err)
	}
	again, ok := byKey.Get()
	if !ok || again.Entity().ID != model.Entity().ID {
		t.Fatalf("expected same model by key, ok=%v model=%+v", ok, again)
	}
}

