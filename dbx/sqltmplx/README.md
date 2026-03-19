# sqltmplx

Primary documentation now lives in Hugo:

- Overview: `/docs/sqltmplx`
- Examples: `/docs/sqltmplx/examples`

Package layout:

- Core: `github.com/DaiYuANg/arcgo/dbx/sqltmplx`
- Dialects: `github.com/DaiYuANg/arcgo/dbx/dialect/{mysql,postgres,sqlite}`
- Validator contract: `github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate`
- Optional validator backends:
  - `github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate/mysqlparser`
  - `github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate/postgresparser`
  - `github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate/sqliteparser`

Quick example:

```go
engine := sqltmplx.New(
    postgres.Dialect{},
    sqltmplx.WithValidator(validate.NewSQLParser(postgres.Dialect{})),
)
```

Import the backend package you want to register:

```go
import _ "github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate/postgresparser"
```
