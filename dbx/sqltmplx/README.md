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

## Statement reuse

**Reuse statements via `MustStatement` or `Statement` to avoid repeated parsing.** The Registry caches compiled templates by name. Obtain the statement once (e.g. at init or first use), then pass it to `dbx.SQLList`, `dbx.SQLGet`, `session.SQL().Exec`, etc. in hot paths:

```go
// Good: build once, execute many
stmt := registry.MustStatement("sql/user/find_active.sql")
for range batches {
    items, _ := dbx.SQLList(ctx, session, stmt, params, mapper)
    // ...
}

// Avoid: parsing on every call
for range batches {
    items, _ := dbx.SQLList(ctx, session, registry.MustStatement("sql/user/find_active.sql"), params, mapper)
}
```

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
