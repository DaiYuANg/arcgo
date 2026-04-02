// Package main demonstrates dbx ID generation strategies.
package main

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/dbx"
)

type snowflakeUser struct {
	ID   int64  `dbx:"id"`
	Name string `dbx:"name"`
}

type snowflakeUserSchema struct {
	dbx.Schema[snowflakeUser]
	ID   dbx.IDColumn[snowflakeUser, int64, dbx.IDSnowflake] `dbx:"id,pk"`
	Name dbx.Column[snowflakeUser, string]                   `dbx:"name"`
}

type uuidUser struct {
	ID   string `dbx:"id"`
	Name string `dbx:"name"`
}

type uuidUserSchema struct {
	dbx.Schema[uuidUser]
	ID   dbx.Column[uuidUser, string] `dbx:"id,pk"`
	Name dbx.Column[uuidUser, string] `dbx:"name"`
}

type strongTypedUser struct {
	ID   int64  `dbx:"id"`
	Name string `dbx:"name"`
}

type strongTypedUserSchema struct {
	dbx.Schema[strongTypedUser]
	ID   dbx.IDColumn[strongTypedUser, int64, dbx.IDSnowflake] `dbx:"id,pk"`
	Name dbx.Column[strongTypedUser, string]                   `dbx:"name"`
}

func main() {
	snowflakeSchema := dbx.MustSchema("snowflake_users", snowflakeUserSchema{})
	idGenerator, err := dbx.NewSnowflakeGenerator(dbx.DefaultNodeID)
	if err != nil {
		panic(err)
	}
	snowflakeEntity := &snowflakeUser{Name: "alice"}
	snowflakeAssignments, err := dbx.MustMapper[snowflakeUser](snowflakeSchema).InsertAssignmentsWithID(context.Background(), snowflakeSchema, snowflakeEntity, idGenerator)
	if err != nil {
		panic(err)
	}

	uuidSchema := dbx.MustSchema("uuid_users", uuidUserSchema{})
	uuidEntity := &uuidUser{Name: "bob"}
	uuidAssignments, err := dbx.MustMapper[uuidUser](uuidSchema).InsertAssignmentsWithID(context.Background(), uuidSchema, uuidEntity, idGenerator)
	if err != nil {
		panic(err)
	}

	strongTypedSchema := dbx.MustSchema("strong_typed_users", strongTypedUserSchema{})
	strongTypedEntity := &strongTypedUser{Name: "carol"}
	strongTypedAssignments, err := dbx.MustMapper[strongTypedUser](strongTypedSchema).InsertAssignmentsWithID(context.Background(), strongTypedSchema, strongTypedEntity, idGenerator)
	if err != nil {
		panic(err)
	}

	printLine("Snowflake by marker type:")
	printFormat("- strategy=%s generated_id=%d assignments=%d\n", snowflakeSchema.ID.Meta().IDStrategy, snowflakeEntity.ID, snowflakeAssignments.Len())

	printLine("UUID by default (string pk => uuidv7):")
	printFormat("- strategy=%s uuid_version=%s generated_id=%s assignments=%d\n", uuidSchema.ID.Meta().IDStrategy, uuidSchema.ID.Meta().UUIDVersion, uuidEntity.ID, uuidAssignments.Len())

	printLine("Snowflake by typed IDColumn marker:")
	printFormat("- strategy=%s generated_id=%d assignments=%d\n", strongTypedSchema.ID.Meta().IDStrategy, strongTypedEntity.ID, strongTypedAssignments.Len())
}

func printLine(text string) {
	if _, err := fmt.Println(text); err != nil {
		panic(err)
	}
}

func printFormat(format string, args ...any) {
	if _, err := fmt.Printf(format, args...); err != nil {
		panic(err)
	}
}
