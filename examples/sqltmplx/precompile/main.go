package main

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/dbx/dialect/mysql"
	"github.com/DaiYuANg/arcgo/dbx/sqltmplx"
)

func main() {
	engine := sqltmplx.New(mysql.Dialect{})
	tpl, err := engine.Compile(`
SELECT id, order_no, status
FROM orders
/*%where */
/*%if present(Tenant) */
  AND tenant = /* Tenant */'acme'
/*%end */
/*%if present(Status) */
  AND status = /* Status */'PAID'
/*%end */
/*%end */
ORDER BY id DESC
`)
	if err != nil {
		panic(err)
	}

	for _, params := range []map[string]any{
		{"Tenant": "acme", "Status": "PAID"},
		{"Tenant": "acme", "Status": "SHIPPED"},
	} {
		bound, err := tpl.Render(params)
		if err != nil {
			panic(err)
		}

		fmt.Println(bound.Query)
		fmt.Println(bound.Args)
	}
}
