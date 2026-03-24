package main

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/dbx"
)

type SnowflakeUser struct {
	ID   int64  `dbx:"id"`
	Name string `dbx:"name"`
}

type SnowflakeUserSchema struct {
	dbx.Schema[SnowflakeUser]
	ID   dbx.IDColumn[SnowflakeUser, int64, dbx.IDSnowflake] `dbx:"id,pk"`
	Name dbx.Column[SnowflakeUser, string] `dbx:"name"`
}

type UUIDUser struct {
	ID   string `dbx:"id"`
	Name string `dbx:"name"`
}

type UUIDUserSchema struct {
	dbx.Schema[UUIDUser]
	ID   dbx.Column[UUIDUser, string] `dbx:"id,pk"`
	Name dbx.Column[UUIDUser, string] `dbx:"name"`
}

type StrongTypedUser struct {
	ID   int64  `dbx:"id"`
	Name string `dbx:"name"`
}

type StrongTypedUserSchema struct {
	dbx.Schema[StrongTypedUser]
	ID   dbx.IDColumn[StrongTypedUser, int64, dbx.IDSnowflake] `dbx:"id,pk"`
	Name dbx.Column[StrongTypedUser, string] `dbx:"name"`
}

func main() {
	snowflakeSchema := dbx.MustSchema("snowflake_users", SnowflakeUserSchema{})
	idGenerator, err := dbx.NewSnowflakeGenerator(dbx.DefaultNodeID)
	if err != nil {
		panic(err)
	}
	snowflakeUser := &SnowflakeUser{Name: "alice"}
	snowflakeAssignments, err := dbx.MustMapper[SnowflakeUser](snowflakeSchema).InsertAssignmentsWithID(context.Background(), snowflakeSchema, snowflakeUser, idGenerator)
	if err != nil {
		panic(err)
	}

	uuidSchema := dbx.MustSchema("uuid_users", UUIDUserSchema{})
	uuidUser := &UUIDUser{Name: "bob"}
	uuidAssignments, err := dbx.MustMapper[UUIDUser](uuidSchema).InsertAssignmentsWithID(context.Background(), uuidSchema, uuidUser, idGenerator)
	if err != nil {
		panic(err)
	}

	strongTypedSchema := dbx.MustSchema("strong_typed_users", StrongTypedUserSchema{})
	strongTypedUser := &StrongTypedUser{Name: "carol"}
	strongTypedAssignments, err := dbx.MustMapper[StrongTypedUser](strongTypedSchema).InsertAssignmentsWithID(context.Background(), strongTypedSchema, strongTypedUser, idGenerator)
	if err != nil {
		panic(err)
	}

	fmt.Println("Snowflake by marker type:")
	fmt.Printf("- strategy=%s generated_id=%d assignments=%d\n", snowflakeSchema.ID.Meta().IDStrategy, snowflakeUser.ID, len(snowflakeAssignments))

	fmt.Println("UUID by default (string pk => uuidv7):")
	fmt.Printf("- strategy=%s uuid_version=%s generated_id=%s assignments=%d\n", uuidSchema.ID.Meta().IDStrategy, uuidSchema.ID.Meta().UUIDVersion, uuidUser.ID, len(uuidAssignments))

	fmt.Println("Snowflake by typed IDColumn marker:")
	fmt.Printf("- strategy=%s generated_id=%d assignments=%d\n", strongTypedSchema.ID.Meta().IDStrategy, strongTypedUser.ID, len(strongTypedAssignments))
}
