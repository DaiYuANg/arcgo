package render

import (
	"testing"

	"github.com/DaiYuANg/arcgo/dbx/sqltmplx/dialect/postgres"
)

type benchmarkNestedFilter struct {
	IDs []int `json:"ids"`
}

type benchmarkParams struct {
	Tenant string                `db:"tenant"`
	Status string                `json:"status"`
	Filter benchmarkNestedFilter `json:"filter"`
}

var benchmarkLookupParams = benchmarkParams{
	Tenant: "acme",
	Status: "active",
	Filter: benchmarkNestedFilter{IDs: []int{1, 2, 3}},
}

func BenchmarkLookupStruct(b *testing.B) {
	for b.Loop() {
		if value := lookup(benchmarkLookupParams, "tenant"); value.IsAbsent() {
			b.Fatal("expected tenant")
		}
	}
}

func BenchmarkLookupNestedStruct(b *testing.B) {
	for b.Loop() {
		if value := lookup(benchmarkLookupParams, "filter.ids"); value.IsAbsent() {
			b.Fatal("expected ids")
		}
	}
}

func BenchmarkEnvMapStruct(b *testing.B) {
	for b.Loop() {
		if env := envMap(benchmarkLookupParams); len(env) == 0 {
			b.Fatal("expected env")
		}
	}
}

func BenchmarkBindTextStruct(b *testing.B) {
	text := "tenant = /* tenant */'acme' AND id IN (/* filter.ids */(1, 2, 3))"

	for b.Loop() {
		st := newState(benchmarkLookupParams, postgres.Dialect{})
		if _, err := bindText(text, st); err != nil {
			b.Fatal(err)
		}
	}
}
