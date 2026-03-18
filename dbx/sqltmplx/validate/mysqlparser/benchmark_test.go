package mysqlparser

import "testing"

func BenchmarkValidateSelect(b *testing.B) {
	parser := New()
	query := "SELECT 1"

	for b.Loop() {
		if err := parser.Validate(query); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAnalyzeSelect(b *testing.B) {
	parser := New()
	query := "SELECT 1"

	for b.Loop() {
		if _, err := parser.Analyze(query); err != nil {
			b.Fatal(err)
		}
	}
}
