package dbx

import (
	"database/sql"
	"errors"
	"testing"
)

type Role struct {
	ID   int64  `dbx:"id"`
	Name string `dbx:"name"`
}

type User struct {
	ID       int64  `dbx:"id"`
	Username string `dbx:"username"`
	Email    string `dbx:"email_address"`
	Status   int    `dbx:"status"`
	RoleID   int64  `dbx:"role_id"`
	Ignored  string `dbx:"ignored"`
}

type UUIDAccount struct {
	ID   string `dbx:"id"`
	Name string `dbx:"name"`
}

type UUIDAccountSchema struct {
	Schema[UUIDAccount]
	ID   Column[UUIDAccount, string] `dbx:"id,pk"`
	Name Column[UUIDAccount, string] `dbx:"name"`
}

type StrongTypedIDUser struct {
	ID   int64  `dbx:"id"`
	Name string `dbx:"name"`
}

type StrongTypedIDUserSchema struct {
	Schema[StrongTypedIDUser]
	ID   IDColumn[StrongTypedIDUser, int64, IDSnowflake] `dbx:"id,pk"`
	Name Column[StrongTypedIDUser, string] `dbx:"name"`
}

type RoleSchema struct {
	Schema[Role]
	ID   Column[Role, int64]  `dbx:"id,pk"`
	Name Column[Role, string] `dbx:"name,unique"`
}

type UserSchema struct {
	Schema[User]
	ID       Column[User, int64] `dbx:"id,pk"`
	Username Column[User, string]
	Email    Column[User, string] `dbx:"email_address,index"`
	Status   Column[User, int]
	RoleID   Column[User, int64]    `dbx:"role_id,ref=roles.id,ondelete=cascade"`
	Role     BelongsTo[User, Role]  `rel:"table=roles,local=role_id,target=id"`
	Peer     HasOne[User, User]     `rel:"table=user_peers,local=id,target=user_id"`
	Children HasMany[User, User]    `rel:"table=users,local=id,target=parent_id"`
	Roles    ManyToMany[User, Role] `rel:"table=roles,target=id,join=user_roles,join_local=user_id,join_target=role_id"`
}

func TestMustSchemaBindsColumnsAndRelations(t *testing.T) {
	users := MustSchema("users", UserSchema{})

	if users.TableName() != "users" {
		t.Fatalf("unexpected table name: %q", users.TableName())
	}
	if users.ID.Ref() != "users.id" {
		t.Fatalf("unexpected id ref: %q", users.ID.Ref())
	}
	if users.Email.Name() != "email_address" {
		t.Fatalf("unexpected email column name: %q", users.Email.Name())
	}
	if !users.ID.IsPrimaryKey() || !users.ID.Meta().AutoIncrement {
		t.Fatalf("expected id metadata to mark pk/auto: %+v", users.ID.Meta())
	}
	if users.ID.Meta().IDStrategy != IDStrategyDBAuto {
		t.Fatalf("expected default int64 id strategy db_auto, got: %q", users.ID.Meta().IDStrategy)
	}
	ref, ok := users.RoleID.Reference()
	if !ok {
		t.Fatal("expected role_id reference metadata")
	}
	if ref.TargetTable != "roles" || ref.TargetColumn != "id" || ref.OnDelete != ReferentialCascade {
		t.Fatalf("unexpected reference metadata: %+v", ref)
	}

	columns := users.Columns()
	if len(columns) != 5 {
		t.Fatalf("unexpected columns metadata count: %d", len(columns))
	}
	relations := users.Relations()
	if len(relations) != 4 {
		t.Fatalf("unexpected relations metadata count: %d", len(relations))
	}
	if relations[0].Kind != RelationBelongsTo || relations[0].TargetTable != "roles" {
		t.Fatalf("unexpected first relation metadata: %+v", relations[0])
	}
	if relations[3].Kind != RelationManyToMany || relations[3].ThroughTable != "user_roles" {
		t.Fatalf("unexpected many-to-many metadata: %+v", relations[3])
	}

	foreignKeys := users.ForeignKeys()
	if len(foreignKeys) != 1 {
		t.Fatalf("unexpected foreign key count: %d", len(foreignKeys))
	}
	if foreignKeys[0].Columns[0] != "role_id" || foreignKeys[0].TargetTable != "roles" {
		t.Fatalf("unexpected foreign key metadata: %+v", foreignKeys[0])
	}
}

func TestAliasRebindsSchemaColumns(t *testing.T) {
	users := MustSchema("users", UserSchema{})
	aliased := Alias(users, "u")

	if aliased.Alias() != "u" {
		t.Fatalf("unexpected alias: %q", aliased.Alias())
	}
	if aliased.ID.Ref() != "u.id" {
		t.Fatalf("unexpected aliased id ref: %q", aliased.ID.Ref())
	}
	if aliased.Email.Ref() != "u.email_address" {
		t.Fatalf("unexpected aliased email ref: %q", aliased.Email.Ref())
	}
}

func TestDefaultUUIDIDStrategyForStringPrimaryKey(t *testing.T) {
	accounts := MustSchema("accounts", UUIDAccountSchema{})
	meta := accounts.ID.Meta()
	if meta.IDStrategy != IDStrategyUUID {
		t.Fatalf("expected uuid id strategy, got %q", meta.IDStrategy)
	}
	if meta.UUIDVersion != DefaultUUIDVersion {
		t.Fatalf("expected default uuid version %q, got %q", DefaultUUIDVersion, meta.UUIDVersion)
	}
	if meta.AutoIncrement {
		t.Fatalf("expected uuid id to disable auto increment: %+v", meta)
	}
}

func TestIDColumnTypeAppliesIDStrategy(t *testing.T) {
	schema := MustSchema("users", StrongTypedIDUserSchema{})
	meta := schema.ID.Meta()
	if meta.IDStrategy != IDStrategySnowflake {
		t.Fatalf("expected snowflake id strategy from typed id column, got %q", meta.IDStrategy)
	}
	if meta.AutoIncrement {
		t.Fatalf("snowflake strategy should disable auto increment: %+v", meta)
	}
}

func TestMustMapperBuildsEntityMappingOnly(t *testing.T) {
	users := MustSchema("users", UserSchema{})
	mapper := MustMapper[User](users)

	fields := mapper.Fields()
	if len(fields) != 5 {
		t.Fatalf("unexpected mapped fields count: %d", len(fields))
	}
	field, ok := mapper.FieldByColumn("role_id")
	if !ok || field.Name != "RoleID" {
		t.Fatalf("unexpected mapper field lookup: %+v %v", field, ok)
	}
}

func TestSelectAndMutationBuilders(t *testing.T) {
	users := MustSchema("users", UserSchema{})
	query := Select(users.ID, users.Username).
		From(users).
		Where(And(users.Status.Eq(1), Like(users.Username, "a%"))).
		OrderBy(users.ID.Desc()).
		Limit(20).
		Offset(10)

	if query.FromItem.Name() != "users" {
		t.Fatalf("unexpected from table: %q", query.FromItem.Name())
	}
	if len(query.Items) != 2 {
		t.Fatalf("unexpected select items: %d", len(query.Items))
	}
	if len(query.Orders) != 1 {
		t.Fatalf("unexpected orders: %d", len(query.Orders))
	}

	insert := InsertInto(users).Values(users.Username.Set("alice"), users.Status.Set(1))
	if len(insert.Assignments) != 2 {
		t.Fatalf("unexpected insert assignments: %d", len(insert.Assignments))
	}

	update := Update(users).Set(users.Status.Set(2)).Where(users.ID.Eq(10))
	if len(update.Assignments) != 1 || update.WhereExp == nil {
		t.Fatalf("unexpected update query state: %+v", update)
	}

	deleteQuery := DeleteFrom(users).Where(users.ID.Eq(10))
	if deleteQuery.WhereExp == nil {
		t.Fatal("expected delete predicate")
	}
}

func TestQueryBuildersCompactNilInputs(t *testing.T) {
	users := MustSchema("users", UserSchema{})

	query := Select(users.ID, nil).
		From(users).
		OrderBy(nil, users.ID.Desc())
	if len(query.Items) != 1 {
		t.Fatalf("unexpected select items after nil compaction: %d", len(query.Items))
	}
	if len(query.Orders) != 1 {
		t.Fatalf("unexpected orders after nil compaction: %d", len(query.Orders))
	}

	insert := InsertInto(users).Values(nil, users.Username.Set("alice"))
	if len(insert.Assignments) != 1 {
		t.Fatalf("unexpected insert assignments after nil compaction: %d", len(insert.Assignments))
	}

	update := Update(users).Set(nil, users.Status.Set(1))
	if len(update.Assignments) != 1 {
		t.Fatalf("unexpected update assignments after nil compaction: %d", len(update.Assignments))
	}
}

func TestOptionsPresets(t *testing.T) {
	db, err := NewWithOptions((*sql.DB)(nil), testSQLiteDialect{}, TestOptions()...)
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	if !db.Debug() {
		t.Error("TestOptions should enable debug")
	}
	db, err = NewWithOptions((*sql.DB)(nil), testSQLiteDialect{}, ProductionOptions()...)
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	if db.Debug() {
		t.Error("ProductionOptions should disable debug")
	}
	db, err = NewWithOptions((*sql.DB)(nil), testSQLiteDialect{}, DefaultOptions()...)
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	if db.Debug() {
		t.Error("DefaultOptions should have debug false")
	}
}

func TestDBWrapper(t *testing.T) {
	core := New((*sql.DB)(nil), testSQLiteDialect{})
	bound := core.Bound("select 1 where id = ?", 1)
	if bound.SQL != "select 1 where id = ?" || len(bound.Args) != 1 {
		t.Fatalf("unexpected bound query: %+v", bound)
	}
}

func TestNewWithOptionsRejectsConflictingIDOptions(t *testing.T) {
	generator, err := NewSnowflakeGenerator(DefaultNodeID)
	if err != nil {
		t.Fatalf("NewSnowflakeGenerator returned error: %v", err)
	}
	_, err = NewWithOptions((*sql.DB)(nil), testSQLiteDialect{},
		WithIDGenerator(generator),
		WithNodeID(DefaultNodeID),
	)
	if !errors.Is(err, ErrIDGeneratorNodeIDConflict) {
		t.Fatalf("expected ErrIDGeneratorNodeIDConflict, got: %v", err)
	}
}

func TestNewWithOptionsRejectsInvalidNodeID(t *testing.T) {
	_, err := NewWithOptions((*sql.DB)(nil), testSQLiteDialect{}, WithNodeID(0))
	if err == nil {
		t.Fatal("expected invalid node id error")
	}
	if !errors.Is(err, ErrInvalidNodeID) {
		t.Fatalf("expected ErrInvalidNodeID, got: %v", err)
	}
	var outOfRange *NodeIDOutOfRangeError
	if !errors.As(err, &outOfRange) {
		t.Fatalf("expected NodeIDOutOfRangeError, got: %T", err)
	}
	if outOfRange.NodeID != 0 {
		t.Fatalf("unexpected node id in error: %d", outOfRange.NodeID)
	}
}
