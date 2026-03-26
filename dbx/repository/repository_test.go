package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DaiYuANg/arcgo/dbx"
	sqlitedialect "github.com/DaiYuANg/arcgo/dbx/dialect/sqlite"
	_ "modernc.org/sqlite"
)

type User struct {
	ID   int64  `dbx:"id"`
	Name string `dbx:"name"`
}

type UserSchema struct {
	dbx.Schema[User]
	ID   dbx.Column[User, int64]  `dbx:"id,pk,auto"`
	Name dbx.Column[User, string] `dbx:"name"`
}

type Device struct {
	DeviceID string `dbx:"device_id"`
	Name     string `dbx:"name"`
}

type DeviceSchema struct {
	dbx.Schema[Device]
	DeviceID dbx.Column[Device, string] `dbx:"device_id,pk"`
	Name     dbx.Column[Device, string] `dbx:"name"`
}

type Membership struct {
	TenantID int64  `dbx:"tenant_id"`
	UserID   int64  `dbx:"user_id"`
	Role     string `dbx:"role"`
}

type MembershipSchema struct {
	dbx.Schema[Membership]
	TenantID dbx.Column[Membership, int64]  `dbx:"tenant_id"`
	UserID   dbx.Column[Membership, int64]  `dbx:"user_id"`
	Role     dbx.Column[Membership, string] `dbx:"role"`
	PK       dbx.CompositeKey[Membership]   `key:"columns=tenant_id|user_id"`
}

type VersionedUser struct {
	ID      int64  `dbx:"id"`
	Name    string `dbx:"name"`
	Version int64  `dbx:"version"`
}

type VersionedUserSchema struct {
	dbx.Schema[VersionedUser]
	ID      dbx.Column[VersionedUser, int64]  `dbx:"id,pk,auto"`
	Name    dbx.Column[VersionedUser, string] `dbx:"name"`
	Version dbx.Column[VersionedUser, int64]  `dbx:"version,default=1"`
}

func TestNewUsesSchemaAsMetadataSource(t *testing.T) {
	core := dbx.New((*sql.DB)(nil), sqlitedialect.New())
	users := dbx.MustSchema("users", UserSchema{})
	repo := New[User](core, users)

	if repo.DB() != core {
		t.Fatal("expected repository to hold db core")
	}
	if repo.Schema().TableName() != "users" {
		t.Fatalf("unexpected schema table: %q", repo.Schema().TableName())
	}
	if _, ok := repo.Mapper().FieldByColumn("name"); !ok {
		t.Fatal("expected mapper to expose name column")
	}
}

func TestBaseCreateListAndFirst(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:repository_crud_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer raw.Close()
	core := dbx.MustNewWithOptions(raw, sqlitedialect.New())
	users := dbx.MustSchema("users", UserSchema{})
	if _, err := core.AutoMigrate(ctx, users); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	repo := New[User](core, users)

	if err := repo.Create(ctx, &User{Name: "alice"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	items, err := repo.List(ctx, nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 || items[0].Name != "alice" {
		t.Fatalf("unexpected items: %+v", items)
	}
	item, err := repo.First(ctx, dbx.Select(users.AllColumns()...).From(users).Where(users.Name.Eq("alice")))
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if item.Name != "alice" {
		t.Fatalf("unexpected first item: %+v", item)
	}
}

func TestBaseFirstNotFound(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:repository_not_found_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer raw.Close()
	core := dbx.MustNewWithOptions(raw, sqlitedialect.New())
	users := dbx.MustSchema("users", UserSchema{})
	if _, err := core.AutoMigrate(ctx, users); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	repo := New[User](core, users)
	_, err = repo.First(ctx, dbx.Select(users.AllColumns()...).From(users).Where(users.Name.Eq("nobody")))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestBaseGetByIDCountExistsUpdateDeleteByIDAndListPage(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:repository_features_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer raw.Close()
	core := dbx.MustNewWithOptions(raw, sqlitedialect.New())
	users := dbx.MustSchema("users", UserSchema{})
	if _, err := core.AutoMigrate(ctx, users); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	repo := New[User](core, users)

	if err := repo.Create(ctx, &User{Name: "alice"}); err != nil {
		t.Fatalf("create alice: %v", err)
	}
	if err := repo.Create(ctx, &User{Name: "bob"}); err != nil {
		t.Fatalf("create bob: %v", err)
	}

	total, err := repo.Count(ctx, nil)
	if err != nil || total != 2 {
		t.Fatalf("count got total=%d err=%v", total, err)
	}
	exists, err := repo.Exists(ctx, dbx.Select(users.AllColumns()...).From(users).Where(users.Name.Eq("alice")))
	if err != nil || !exists {
		t.Fatalf("exists got exists=%v err=%v", exists, err)
	}

	alice, err := repo.First(ctx, dbx.Select(users.AllColumns()...).From(users).Where(users.Name.Eq("alice")))
	if err != nil {
		t.Fatalf("first alice: %v", err)
	}
	got, err := repo.GetByID(ctx, alice.ID)
	if err != nil || got.Name != "alice" {
		t.Fatalf("get by id got=%+v err=%v", got, err)
	}

	if _, err := repo.UpdateByID(ctx, alice.ID, users.Name.Set("alice-updated")); err != nil {
		t.Fatalf("update by id: %v", err)
	}
	updated, err := repo.GetByID(ctx, alice.ID)
	if err != nil || updated.Name != "alice-updated" {
		t.Fatalf("updated get got=%+v err=%v", updated, err)
	}

	page, err := repo.ListPage(ctx, dbx.Select(users.AllColumns()...).From(users).OrderBy(users.Name.Asc()), 1, 1)
	if err != nil {
		t.Fatalf("list page: %v", err)
	}
	if page.Total != 2 || page.Page != 1 || page.PageSize != 1 || len(page.Items) != 1 {
		t.Fatalf("unexpected page result: %+v", page)
	}

	if _, err := repo.DeleteByID(ctx, alice.ID); err != nil {
		t.Fatalf("delete by id: %v", err)
	}
	afterDelete, err := repo.Count(ctx, nil)
	if err != nil || afterDelete != 1 {
		t.Fatalf("count after delete total=%d err=%v", afterDelete, err)
	}
}

func TestBaseByIDUsesPrimaryKeyColumnFromSchema(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:repository_pk_column_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer raw.Close()
	core := dbx.MustNewWithOptions(raw, sqlitedialect.New())
	devices := dbx.MustSchema("devices", DeviceSchema{})
	if _, err := core.AutoMigrate(ctx, devices); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	repo := New[Device](core, devices)

	if err := repo.Create(ctx, &Device{DeviceID: "dev-1", Name: "sensor"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	item, err := repo.GetByID(ctx, "dev-1")
	if err != nil || item.Name != "sensor" {
		t.Fatalf("get by pk got=%+v err=%v", item, err)
	}
	if _, err := repo.UpdateByID(ctx, "dev-1", devices.Name.Set("sensor-v2")); err != nil {
		t.Fatalf("update by pk: %v", err)
	}
	updated, err := repo.GetByID(ctx, "dev-1")
	if err != nil || updated.Name != "sensor-v2" {
		t.Fatalf("updated by pk got=%+v err=%v", updated, err)
	}
	if _, err := repo.DeleteByID(ctx, "dev-1"); err != nil {
		t.Fatalf("delete by pk: %v", err)
	}
	if _, err := repo.GetByID(ctx, "dev-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestBaseByIDNotFoundAsErrorOption(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:repository_not_found_option_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer raw.Close()
	core := dbx.MustNewWithOptions(raw, sqlitedialect.New())
	users := dbx.MustSchema("users", UserSchema{})
	if _, err := core.AutoMigrate(ctx, users); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	defaultRepo := New[User](core, users)
	if _, err := defaultRepo.DeleteByID(ctx, int64(404)); err != nil {
		t.Fatalf("default delete should not error on not found: %v", err)
	}
	if _, err := defaultRepo.UpdateByID(ctx, int64(404), users.Name.Set("missing")); err != nil {
		t.Fatalf("default update should not error on not found: %v", err)
	}

	strictRepo := NewWithOptions[User](core, users, WithByIDNotFoundAsError(true))
	if _, err := strictRepo.DeleteByID(ctx, int64(404)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("strict delete expected ErrNotFound, got: %v", err)
	}
	if _, err := strictRepo.UpdateByID(ctx, int64(404), users.Name.Set("missing")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("strict update expected ErrNotFound, got: %v", err)
	}
}

func TestBaseCreateManyAndUpsert(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:repository_create_many_upsert_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer raw.Close()
	core := dbx.MustNewWithOptions(raw, sqlitedialect.New())
	users := dbx.MustSchema("users", UserSchema{})
	if _, err := core.AutoMigrate(ctx, users); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	repo := New[User](core, users)

	if err := repo.CreateMany(ctx, &User{Name: "alice"}, &User{Name: "bob"}); err != nil {
		t.Fatalf("create many: %v", err)
	}
	total, err := repo.Count(ctx, nil)
	if err != nil || total != 2 {
		t.Fatalf("count after create many total=%d err=%v", total, err)
	}

	devices := dbx.MustSchema("devices", DeviceSchema{})
	if _, err := core.AutoMigrate(ctx, devices); err != nil {
		t.Fatalf("auto migrate devices: %v", err)
	}
	deviceRepo := New[Device](core, devices)
	if err := deviceRepo.Create(ctx, &Device{DeviceID: "dev-1", Name: "sensor"}); err != nil {
		t.Fatalf("seed device: %v", err)
	}
	if err := deviceRepo.Upsert(ctx, &Device{DeviceID: "dev-1", Name: "sensor-v2"}); err != nil {
		t.Fatalf("upsert by pk: %v", err)
	}
	first, err := deviceRepo.GetByID(ctx, "dev-1")
	if err != nil || first.Name != "sensor-v2" {
		t.Fatalf("upsert result got=%+v err=%v", first, err)
	}
}

func TestBaseCompositePrimaryKeyByKey(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:repository_composite_key_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer raw.Close()
	core := dbx.MustNewWithOptions(raw, sqlitedialect.New())
	memberships := dbx.MustSchema("memberships", MembershipSchema{})
	if _, err := core.AutoMigrate(ctx, memberships); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	repo := New[Membership](core, memberships)

	if err := repo.Create(ctx, &Membership{TenantID: 100, UserID: 200, Role: "viewer"}); err != nil {
		t.Fatalf("create membership: %v", err)
	}

	key := Key{"tenant_id": int64(100), "user_id": int64(200)}
	item, err := repo.GetByKey(ctx, key)
	if err != nil || item.Role != "viewer" {
		t.Fatalf("get by key got=%+v err=%v", item, err)
	}

	if _, err := repo.UpdateByKey(ctx, key, memberships.Role.Set("admin")); err != nil {
		t.Fatalf("update by key: %v", err)
	}
	updated, err := repo.GetByKey(ctx, key)
	if err != nil || updated.Role != "admin" {
		t.Fatalf("updated by key got=%+v err=%v", updated, err)
	}
	if _, err := repo.DeleteByKey(ctx, key); err != nil {
		t.Fatalf("delete by key: %v", err)
	}
	_, err = repo.GetByKey(ctx, key)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete by key, got: %v", err)
	}
}

func TestBaseSpecAPIs(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:repository_spec_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer raw.Close()
	core := dbx.MustNewWithOptions(raw, sqlitedialect.New())
	users := dbx.MustSchema("users", UserSchema{})
	if _, err := core.AutoMigrate(ctx, users); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	repo := New[User](core, users)
	if err := repo.CreateMany(ctx, &User{Name: "alice"}, &User{Name: "bob"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	items, err := repo.ListSpec(ctx, Where(users.Name.Eq("alice")))
	if err != nil || len(items) != 1 {
		t.Fatalf("list spec items=%d err=%v", len(items), err)
	}
	exists, err := repo.ExistsSpec(ctx, Where(users.Name.Eq("alice")))
	if err != nil || !exists {
		t.Fatalf("exists spec exists=%v err=%v", exists, err)
	}
	total, err := repo.CountSpec(ctx, Where(users.Name.Eq("alice")))
	if err != nil || total != 1 {
		t.Fatalf("count spec total=%d err=%v", total, err)
	}
	page, err := repo.ListPageSpec(ctx, 1, 1, OrderBy(users.Name.Asc()))
	if err != nil || page.Total != 2 || len(page.Items) != 1 {
		t.Fatalf("list page spec result=%+v err=%v", page, err)
	}
}

func TestBaseOptionAPIs(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:repository_option_api_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer raw.Close()
	core := dbx.MustNewWithOptions(raw, sqlitedialect.New())
	users := dbx.MustSchema("users", UserSchema{})
	if _, err := core.AutoMigrate(ctx, users); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	repo := New[User](core, users)
	if err := repo.Create(ctx, &User{Name: "alice"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	noneByID, err := repo.GetByIDOption(ctx, int64(99999))
	if err != nil {
		t.Fatalf("get by id option: %v", err)
	}
	if noneByID.IsPresent() {
		t.Fatal("expected absent option for missing id")
	}

	someBySpec, err := repo.FirstSpecOption(ctx, Where(users.Name.Eq("alice")))
	if err != nil {
		t.Fatalf("first spec option: %v", err)
	}
	item, ok := someBySpec.Get()
	if !ok || item.Name != "alice" {
		t.Fatalf("expected alice from option, got ok=%v item=%+v", ok, item)
	}

	noneBySpec, err := repo.FirstSpecOption(ctx, Where(users.Name.Eq("nobody")))
	if err != nil {
		t.Fatalf("first spec none option: %v", err)
	}
	if noneBySpec.IsPresent() {
		t.Fatal("expected absent option for missing record")
	}
}

func TestBaseUpdateByVersion(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:repository_version_conflict_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer raw.Close()
	core := dbx.MustNewWithOptions(raw, sqlitedialect.New())
	users := dbx.MustSchema("versioned_users", VersionedUserSchema{})
	if _, err := core.AutoMigrate(ctx, users); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	repo := New[VersionedUser](core, users)
	if err := repo.Create(ctx, &VersionedUser{Name: "alice", Version: 1}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	item, err := repo.First(ctx, dbx.Select(users.AllColumns()...).From(users))
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	key := Key{"id": item.ID}
	if _, err := repo.UpdateByVersion(ctx, key, 1, users.Name.Set("alice-v2")); err != nil {
		t.Fatalf("update by version: %v", err)
	}
	if _, err := repo.UpdateByVersion(ctx, key, 1, users.Name.Set("alice-stale")); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got: %v", err)
	}
}

func TestBaseFirstDoesNotMutateQuery(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:repository_first_immutable_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer raw.Close()
	core := dbx.MustNewWithOptions(raw, sqlitedialect.New())
	users := dbx.MustSchema("users", UserSchema{})
	if _, err := core.AutoMigrate(ctx, users); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	repo := New[User](core, users)
	if err := repo.Create(ctx, &User{Name: "alice"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	query := dbx.Select(users.AllColumns()...).From(users).Where(users.Name.Eq("alice"))
	if _, err := repo.First(ctx, query); err != nil {
		t.Fatalf("first: %v", err)
	}
	if query.LimitN != nil {
		t.Fatalf("expected First to leave query limit unchanged, got: %d", *query.LimitN)
	}
	if query.OffsetN != nil {
		t.Fatalf("expected First to leave query offset unchanged, got: %d", *query.OffsetN)
	}
}

func TestBaseListPageDoesNotMutateQuery(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:repository_page_immutable_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer raw.Close()
	core := dbx.MustNewWithOptions(raw, sqlitedialect.New())
	users := dbx.MustSchema("users", UserSchema{})
	if _, err := core.AutoMigrate(ctx, users); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	repo := New[User](core, users)
	if err := repo.CreateMany(ctx, &User{Name: "alice"}, &User{Name: "bob"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	query := dbx.Select(users.AllColumns()...).From(users).OrderBy(users.Name.Asc())
	if _, err := repo.ListPage(ctx, query, 2, 1); err != nil {
		t.Fatalf("list page: %v", err)
	}
	if query.LimitN != nil {
		t.Fatalf("expected ListPage to leave query limit unchanged, got: %d", *query.LimitN)
	}
	if query.OffsetN != nil {
		t.Fatalf("expected ListPage to leave query offset unchanged, got: %d", *query.OffsetN)
	}
}
