package render

import (
	"testing"

	"github.com/DaiYuANg/arcgo/dbx/dialect/postgres"
)

type bindQuery struct {
	Name string `db:"name"`
	IDs  []int  `json:"ids"`
}

func TestBindCommentPlaceholderWithStructTags(t *testing.T) {
	st := newState(bindQuery{Name: "alice", IDs: []int{10, 20}}, postgres.Dialect{})
	out, err := bindText("name = /* name */'bob' AND id IN (/* ids */(1, 2))", st)
	if err != nil {
		t.Fatalf("bindText returned error: %v", err)
	}
	if out != "name = $1 AND id IN ($2, $3)" {
		t.Fatalf("unexpected bind output: %q", out)
	}
	if len(st.args) != 3 {
		t.Fatalf("unexpected args length: %d", len(st.args))
	}
}
