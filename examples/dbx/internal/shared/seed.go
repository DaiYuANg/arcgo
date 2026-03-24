package shared

import (
	"context"

	"github.com/DaiYuANg/arcgo/dbx"
)

func InsertAll[E any, S dbx.SchemaSource[E]](ctx context.Context, session dbx.Session, schema S, items ...E) error {
	mapper := dbx.MustMapper[E](schema)
	for _, item := range items {
		assignments, err := mapper.InsertAssignments(session, schema, new(item))
		if err != nil {
			return err
		}
		if _, err := dbx.Exec(ctx, session, dbx.InsertInto(schema).Values(assignments...)); err != nil {
			return err
		}
	}
	return nil
}

func SeedDemoData(ctx context.Context, session dbx.Session, catalog Catalog) error {
	if err := InsertAll(ctx, session, catalog.Roles,
		Role{Name: "admin"},
		Role{Name: "reader"},
		Role{Name: "auditor"},
	); err != nil {
		return err
	}

	if err := InsertAll(ctx, session, catalog.Users,
		User{Username: "alice", Email: "alice@example.com", Status: 1, RoleID: 1},
		User{Username: "bob", Email: "bob@example.com", Status: 1, RoleID: 2},
		User{Username: "carol", Email: "carol@example.com", Status: 0, RoleID: 3},
	); err != nil {
		return err
	}

	return InsertAll(ctx, session, catalog.UserRoles,
		UserRoleLink{UserID: 1, RoleID: 1},
		UserRoleLink{UserID: 1, RoleID: 2},
		UserRoleLink{UserID: 2, RoleID: 2},
		UserRoleLink{UserID: 3, RoleID: 3},
	)
}
