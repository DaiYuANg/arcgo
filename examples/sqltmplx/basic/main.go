package main

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/dbx/sqltmplx"
	"github.com/DaiYuANg/arcgo/dbx/sqltmplx/dialect/mysql"
	"github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate"
	_ "github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate/mysqlparser"
)

func main() {
	engine := sqltmplx.New(
		mysql.Dialect{},
		sqltmplx.WithValidator(validate.NewSQLParser(mysql.Dialect{})),
	)

	tpl := `
SELECT id, name, status
FROM users
/*%where */
/*%if name != nil */
  AND name = /* name */'alice'
/*%end */
/*%if ids != nil */
  AND id IN (/* ids */(1, 2, 3))
/*%end */
/*%end */
ORDER BY id DESC
`

	bound, err := engine.Render(tpl, map[string]any{
		"name": "alice",
		"ids":  []int{1, 2, 3},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(bound.Query)
	fmt.Println(bound.Args)
}
