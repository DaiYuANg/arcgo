package dbx

import (
	"fmt"
	"reflect"
)

func normalizeIDPolicy(meta ColumnMeta) (ColumnMeta, error) {
	if !meta.PrimaryKey {
		return meta, nil
	}

	columnType := meta.GoType
	for columnType != nil && columnType.Kind() == reflect.Pointer {
		columnType = columnType.Elem()
	}

	if meta.IDStrategy == IDStrategyUnset {
		switch {
		case columnType != nil && columnType.Kind() == reflect.Int64:
			meta.IDStrategy = IDStrategyDBAuto
		case columnType != nil && columnType.Kind() == reflect.String:
			meta.IDStrategy = IDStrategyUUID
			if meta.UUIDVersion == "" {
				meta.UUIDVersion = DefaultUUIDVersion
			}
		}
	}

	switch meta.IDStrategy {
	case IDStrategyUnset:
		return meta, nil
	case IDStrategyDBAuto:
		meta.AutoIncrement = true
		meta.UUIDVersion = ""
	case IDStrategySnowflake:
		if columnType == nil || columnType.Kind() != reflect.Int64 {
			return meta, fmt.Errorf("dbx: snowflake id strategy only supports int64 primary keys, column %s", meta.Name)
		}
		meta.AutoIncrement = false
		meta.UUIDVersion = ""
	case IDStrategyUUID:
		if columnType == nil || columnType.Kind() != reflect.String {
			return meta, fmt.Errorf("dbx: uuid id strategy only supports string primary keys, column %s", meta.Name)
		}
		meta.AutoIncrement = false
		if meta.UUIDVersion == "" {
			meta.UUIDVersion = DefaultUUIDVersion
		}
		if meta.UUIDVersion != "v7" && meta.UUIDVersion != "v4" {
			return meta, fmt.Errorf("dbx: unsupported uuid version %q for column %s", meta.UUIDVersion, meta.Name)
		}
	case IDStrategyULID, IDStrategyKSUID:
		if columnType == nil || columnType.Kind() != reflect.String {
			return meta, fmt.Errorf("dbx: %s id strategy only supports string primary keys, column %s", meta.IDStrategy, meta.Name)
		}
		meta.AutoIncrement = false
		meta.UUIDVersion = ""
	default:
		return meta, fmt.Errorf("dbx: unsupported id strategy %q for column %s", meta.IDStrategy, meta.Name)
	}

	return meta, nil
}
