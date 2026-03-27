package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DaiYuANg/arcgo/dbx"
	sqlitedialect "github.com/DaiYuANg/arcgo/dbx/dialect/sqlite"
	repository "github.com/DaiYuANg/arcgo/dbx/repository"
	"github.com/stretchr/testify/require"
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
	repo := repository.New[User](core, users)

	require.Same(t, core, repo.DB())
	require.Equal(t, "users", repo.Schema().TableName())

	_, ok := repo.Mapper().FieldByColumn("name")
	require.True(t, ok)
}

func TestBaseCreateListAndFirst(t *testing.T) {
	repo, users, ctx := newUserRepo(t, "file:repository_crud_test?mode=memory&cache=shared")
	seedUsers(ctx, t, repo, "alice")

	items, err := repo.List(ctx, nil)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "alice", items[0].Name)

	item, err := repo.First(ctx, dbx.Select(users.AllColumns()...).From(users).Where(users.Name.Eq("alice")))
	require.NoError(t, err)
	require.Equal(t, "alice", item.Name)
}

func TestBaseFirstNotFound(t *testing.T) {
	repo, users, ctx := newUserRepo(t, "file:repository_not_found_test?mode=memory&cache=shared")

	_, err := repo.First(ctx, dbx.Select(users.AllColumns()...).From(users).Where(users.Name.Eq("nobody")))
	require.ErrorIs(t, err, repository.ErrNotFound)
}

func TestBaseGetByIDCountExistsUpdateDeleteByIDAndListPage(t *testing.T) {
	repo, users, ctx := newSeededUserRepo(t, "file:repository_features_test?mode=memory&cache=shared", "alice", "bob")

	total, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)

	exists, err := repo.Exists(ctx, dbx.Select(users.AllColumns()...).From(users).Where(users.Name.Eq("alice")))
	require.NoError(t, err)
	require.True(t, exists)

	alice, err := repo.First(ctx, dbx.Select(users.AllColumns()...).From(users).Where(users.Name.Eq("alice")))
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, alice.ID)
	require.NoError(t, err)
	require.Equal(t, "alice", got.Name)

	_, err = repo.UpdateByID(ctx, alice.ID, users.Name.Set("alice-updated"))
	require.NoError(t, err)

	updated, err := repo.GetByID(ctx, alice.ID)
	require.NoError(t, err)
	require.Equal(t, "alice-updated", updated.Name)

	page, err := repo.ListPage(ctx, dbx.Select(users.AllColumns()...).From(users).OrderBy(users.Name.Asc()), 1, 1)
	require.NoError(t, err)
	require.EqualValues(t, 2, page.Total)
	require.Equal(t, 1, page.Page)
	require.Equal(t, 1, page.PageSize)
	require.Len(t, page.Items, 1)

	_, err = repo.DeleteByID(ctx, alice.ID)
	require.NoError(t, err)

	afterDelete, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.EqualValues(t, 1, afterDelete)
}

func TestBaseByIDUsesPrimaryKeyColumnFromSchema(t *testing.T) {
	repo, devices, ctx := newDeviceRepo(t, "file:repository_pk_column_test?mode=memory&cache=shared")
	require.NoError(t, repo.Create(ctx, &Device{DeviceID: "dev-1", Name: "sensor"}))

	item, err := repo.GetByID(ctx, "dev-1")
	require.NoError(t, err)
	require.Equal(t, "sensor", item.Name)

	_, err = repo.UpdateByID(ctx, "dev-1", devices.Name.Set("sensor-v2"))
	require.NoError(t, err)

	updated, err := repo.GetByID(ctx, "dev-1")
	require.NoError(t, err)
	require.Equal(t, "sensor-v2", updated.Name)

	_, err = repo.DeleteByID(ctx, "dev-1")
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, "dev-1")
	require.ErrorIs(t, err, repository.ErrNotFound)
}

func TestBaseByIDNotFoundAsErrorOption(t *testing.T) {
	ctx := context.Background()
	core := openRepositoryCore(t, "file:repository_not_found_option_test?mode=memory&cache=shared")
	users := dbx.MustSchema("users", UserSchema{})
	mustAutoMigrate(ctx, t, core, users)

	defaultRepo := repository.New[User](core, users)
	_, err := defaultRepo.DeleteByID(ctx, int64(404))
	require.NoError(t, err)
	_, err = defaultRepo.UpdateByID(ctx, int64(404), users.Name.Set("missing"))
	require.NoError(t, err)

	strictRepo := repository.NewWithOptions[User](core, users, repository.WithByIDNotFoundAsError(true))
	_, err = strictRepo.DeleteByID(ctx, int64(404))
	require.ErrorIs(t, err, repository.ErrNotFound)
	_, err = strictRepo.UpdateByID(ctx, int64(404), users.Name.Set("missing"))
	require.ErrorIs(t, err, repository.ErrNotFound)
}

func TestBaseCreateManyAndUpsert(t *testing.T) {
	userRepo, _, userCtx := newUserRepo(t, "file:repository_create_many_users_test?mode=memory&cache=shared")
	require.NoError(t, userRepo.CreateMany(userCtx, &User{Name: "alice"}, &User{Name: "bob"}))

	total, err := userRepo.Count(userCtx, nil)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)

	deviceRepo, _, deviceCtx := newDeviceRepo(t, "file:repository_upsert_devices_test?mode=memory&cache=shared")
	require.NoError(t, deviceRepo.Create(deviceCtx, &Device{DeviceID: "dev-1", Name: "sensor"}))
	require.NoError(t, deviceRepo.Upsert(deviceCtx, &Device{DeviceID: "dev-1", Name: "sensor-v2"}))

	device, err := deviceRepo.GetByID(deviceCtx, "dev-1")
	require.NoError(t, err)
	require.Equal(t, "sensor-v2", device.Name)
}

func TestBaseCompositePrimaryKeyByKey(t *testing.T) {
	repo, memberships, ctx := newMembershipRepo(t, "file:repository_composite_key_test?mode=memory&cache=shared")
	require.NoError(t, repo.Create(ctx, &Membership{TenantID: 100, UserID: 200, Role: "viewer"}))

	key := repository.Key{"tenant_id": int64(100), "user_id": int64(200)}

	item, err := repo.GetByKey(ctx, key)
	require.NoError(t, err)
	require.Equal(t, "viewer", item.Role)

	_, err = repo.UpdateByKey(ctx, key, memberships.Role.Set("admin"))
	require.NoError(t, err)

	updated, err := repo.GetByKey(ctx, key)
	require.NoError(t, err)
	require.Equal(t, "admin", updated.Role)

	_, err = repo.DeleteByKey(ctx, key)
	require.NoError(t, err)

	_, err = repo.GetByKey(ctx, key)
	require.ErrorIs(t, err, repository.ErrNotFound)
}

func TestBaseSpecAPIs(t *testing.T) {
	repo, users, ctx := newSeededUserRepo(t, "file:repository_spec_test?mode=memory&cache=shared", "alice", "bob")

	items, err := repo.ListSpec(ctx, repository.Where(users.Name.Eq("alice")))
	require.NoError(t, err)
	require.Len(t, items, 1)

	exists, err := repo.ExistsSpec(ctx, repository.Where(users.Name.Eq("alice")))
	require.NoError(t, err)
	require.True(t, exists)

	total, err := repo.CountSpec(ctx, repository.Where(users.Name.Eq("alice")))
	require.NoError(t, err)
	require.EqualValues(t, 1, total)

	page, err := repo.ListPageSpec(ctx, 1, 1, repository.OrderBy(users.Name.Asc()))
	require.NoError(t, err)
	require.EqualValues(t, 2, page.Total)
	require.Len(t, page.Items, 1)
}

func TestBaseOptionAPIs(t *testing.T) {
	repo, users, ctx := newSeededUserRepo(t, "file:repository_option_api_test?mode=memory&cache=shared", "alice")

	noneByID, err := repo.GetByIDOption(ctx, int64(99999))
	require.NoError(t, err)
	require.False(t, noneByID.IsPresent())

	someBySpec, err := repo.FirstSpecOption(ctx, repository.Where(users.Name.Eq("alice")))
	require.NoError(t, err)

	item, ok := someBySpec.Get()
	require.True(t, ok)
	require.Equal(t, "alice", item.Name)

	noneBySpec, err := repo.FirstSpecOption(ctx, repository.Where(users.Name.Eq("nobody")))
	require.NoError(t, err)
	require.False(t, noneBySpec.IsPresent())
}

func TestBaseUpdateByVersion(t *testing.T) {
	repo, users, ctx := newVersionedUserRepo(t, "file:repository_version_conflict_test?mode=memory&cache=shared")
	require.NoError(t, repo.Create(ctx, &VersionedUser{Name: "alice", Version: 1}))

	item, err := repo.First(ctx, dbx.Select(users.AllColumns()...).From(users))
	require.NoError(t, err)

	key := repository.Key{"id": item.ID}
	_, err = repo.UpdateByVersion(ctx, key, 1, users.Name.Set("alice-v2"))
	require.NoError(t, err)

	_, err = repo.UpdateByVersion(ctx, key, 1, users.Name.Set("alice-stale"))
	require.ErrorIs(t, err, repository.ErrVersionConflict)
}

func TestBaseFirstDoesNotMutateQuery(t *testing.T) {
	repo, users, ctx := newSeededUserRepo(t, "file:repository_first_immutable_test?mode=memory&cache=shared", "alice")

	query := dbx.Select(users.AllColumns()...).From(users).Where(users.Name.Eq("alice"))
	_, err := repo.First(ctx, query)
	require.NoError(t, err)
	require.Nil(t, query.LimitN)
	require.Nil(t, query.OffsetN)
}

func TestBaseListDoesNotMutateQuery(t *testing.T) {
	repo, users, ctx := newSeededUserRepo(t, "file:repository_list_immutable_test?mode=memory&cache=shared", "alice", "bob")

	query := newOrderedUserQuery(users)
	_, err := repo.List(ctx, query)
	require.NoError(t, err)
	assertOrderedUserQueryUnchanged(t, query)
}

func TestBaseCountDoesNotMutateQuery(t *testing.T) {
	repo, users, ctx := newSeededUserRepo(t, "file:repository_count_immutable_test?mode=memory&cache=shared", "alice", "bob")

	query := newOrderedUserQuery(users)
	_, err := repo.Count(ctx, query)
	require.NoError(t, err)
	assertOrderedUserQueryUnchanged(t, query)
}

func TestBaseListPageDoesNotMutateQuery(t *testing.T) {
	repo, users, ctx := newSeededUserRepo(t, "file:repository_page_immutable_test?mode=memory&cache=shared", "alice", "bob")

	query := dbx.Select(users.AllColumns()...).From(users).OrderBy(users.Name.Asc())
	_, err := repo.ListPage(ctx, query, 2, 1)
	require.NoError(t, err)
	require.Nil(t, query.LimitN)
	require.Nil(t, query.OffsetN)
}

func newUserRepo(t *testing.T, dsn string) (*repository.Base[User, UserSchema], UserSchema, context.Context) {
	t.Helper()

	ctx := context.Background()
	core := openRepositoryCore(t, dsn)
	users := dbx.MustSchema("users", UserSchema{})
	mustAutoMigrate(ctx, t, core, users)

	return repository.New[User](core, users), users, ctx
}

func newDeviceRepo(t *testing.T, dsn string) (*repository.Base[Device, DeviceSchema], DeviceSchema, context.Context) {
	t.Helper()

	ctx := context.Background()
	core := openRepositoryCore(t, dsn)
	devices := dbx.MustSchema("devices", DeviceSchema{})
	mustAutoMigrate(ctx, t, core, devices)

	return repository.New[Device](core, devices), devices, ctx
}

func newMembershipRepo(t *testing.T, dsn string) (*repository.Base[Membership, MembershipSchema], MembershipSchema, context.Context) {
	t.Helper()

	ctx := context.Background()
	core := openRepositoryCore(t, dsn)
	memberships := dbx.MustSchema("memberships", MembershipSchema{})
	mustAutoMigrate(ctx, t, core, memberships)

	return repository.New[Membership](core, memberships), memberships, ctx
}

func newVersionedUserRepo(t *testing.T, dsn string) (*repository.Base[VersionedUser, VersionedUserSchema], VersionedUserSchema, context.Context) {
	t.Helper()

	ctx := context.Background()
	core := openRepositoryCore(t, dsn)
	users := dbx.MustSchema("versioned_users", VersionedUserSchema{})
	mustAutoMigrate(ctx, t, core, users)

	return repository.New[VersionedUser](core, users), users, ctx
}

func newSeededUserRepo(t *testing.T, dsn string, names ...string) (*repository.Base[User, UserSchema], UserSchema, context.Context) {
	t.Helper()

	repo, users, ctx := newUserRepo(t, dsn)
	seedUsers(ctx, t, repo, names...)

	return repo, users, ctx
}

func openRepositoryCore(t *testing.T, dsn string) *dbx.DB {
	t.Helper()

	raw, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)

	t.Cleanup(func() {
		if closeErr := raw.Close(); closeErr != nil {
			t.Errorf("close sqlite: %v", closeErr)
		}
	})

	return dbx.MustNewWithOptions(raw, sqlitedialect.New())
}

func mustAutoMigrate(ctx context.Context, t *testing.T, core *dbx.DB, schemas ...dbx.SchemaResource) {
	t.Helper()

	_, err := core.AutoMigrate(ctx, schemas...)
	require.NoError(t, err)
}

func seedUsers(ctx context.Context, t *testing.T, repo *repository.Base[User, UserSchema], names ...string) {
	t.Helper()

	for _, name := range names {
		require.NoError(t, repo.Create(ctx, &User{Name: name}))
	}
}

func newOrderedUserQuery(users UserSchema) *dbx.SelectQuery {
	return dbx.Select(users.AllColumns()...).From(users).OrderBy(users.Name.Asc()).Limit(10).Offset(5)
}

func assertOrderedUserQueryUnchanged(t *testing.T, query *dbx.SelectQuery) {
	t.Helper()

	require.NotNil(t, query.LimitN)
	require.Equal(t, 10, *query.LimitN)
	require.NotNil(t, query.OffsetN)
	require.Equal(t, 5, *query.OffsetN)
	require.Len(t, query.Orders, 1)
}
