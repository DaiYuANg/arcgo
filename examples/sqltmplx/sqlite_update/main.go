package main

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/dbx/dialect/sqlite"
	"github.com/DaiYuANg/arcgo/dbx/sqltmplx"
)

type UpdateCommand struct {
	ID     int
	Name   string `db:"name"`
	Status string `json:"status"`
}

func main() {
	engine := sqltmplx.New(sqlite.Dialect{})

	tpl := `
UPDATE users
/*%set */
/*%if present(Name) */
  name = /* Name */'alice',
/*%end */
/*%if present(Status) */
  status = /* Status */'active',
/*%end */
/*%end */
WHERE id = /* ID */1
`

	bound, err := engine.Render(tpl, UpdateCommand{
		ID:     42,
		Name:   "alice",
		Status: "active",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(bound.Query)
	fmt.Println(bound.Args)
}
