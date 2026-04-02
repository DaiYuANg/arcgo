package mapping_test

import (
	"testing"

	"github.com/DaiYuANg/arcgo/kvx/mapping"
	"github.com/stretchr/testify/require"
)

func TestEntityMetadataStorageNames(t *testing.T) {
	t.Parallel()

	metadata := &mapping.EntityMetadata{
		KeyField: "ID",
		Fields: map[string]mapping.FieldTag{
			"ID":    {FieldName: "ID", Name: "id"},
			"Name":  {FieldName: "Name", Name: "name"},
			"Email": {FieldName: "Email", Name: "email_address"},
		},
	}

	names := metadata.StorageNames()

	require.Equal(t, 2, names.Len())
	require.ElementsMatch(t, []string{"name", "email_address"}, names.Values())
}

func TestEntityMetadataIndexedNames(t *testing.T) {
	t.Parallel()

	metadata := &mapping.EntityMetadata{
		Fields: map[string]mapping.FieldTag{
			"Email":  {FieldName: "Email", Name: "email_address", Index: true},
			"Status": {FieldName: "Status", Name: "status", Index: true, IndexName: "status_idx"},
		},
		IndexFields: []string{"Email", "Status", "Email", "Missing"},
	}

	names := metadata.IndexedNames()

	require.Equal(t, 3, names.Len())
	require.Equal(t, []string{"email_address", "status_idx", "Missing"}, names.Values())
}
