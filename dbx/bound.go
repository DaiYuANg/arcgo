// Package dbx provides a lightweight typed SQL query builder, mapper, and repository toolkit.
package dbx

import "github.com/DaiYuANg/arcgo/collectionx"

type BoundQuery struct {
	Name         string
	SQL          string
	Args         collectionx.List[any]
	CapacityHint int // when >0, hint for pre-allocating result slice (e.g. from LIMIT)
}
