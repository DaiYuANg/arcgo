package main

import (
	"fmt"

	"github.com/DaiYuANg/archgo/sqltmplx"
	"github.com/DaiYuANg/archgo/sqltmplx/dialect"
	"github.com/DaiYuANg/archgo/sqltmplx/validate"
)

func main() {
	engine := sqltmplx.New(
		dialect.MySQL{},
		sqltmplx.WithValidator(validate.Noop{}),
	)

	tpl := `
SELECT id, name, status
FROM users
/* where */
/* if name != nil */
  AND name = #{name}
/* end */
/* if ids != nil */
  AND id IN (#{ids*})
/* end */
/* end */
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
