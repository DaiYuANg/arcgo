package repository_test

import (
	"context"
	"testing"

	"github.com/DaiYuANg/arcgo/kvx/repository"
)

func TestHashRepository_FindAll_ScansAllPagesAndDeduplicates(t *testing.T) {
	ctx := context.Background()
	hash := newMockHash()
	kv := newMockKV()
	kv.scanPages = [][]string{
		{"user:user1", "user:user2"},
		{"user:user2", "user:user3"},
	}
	repo := repository.NewHashRepository[TestUser](hash, kv, "user")

	hash.data["user:user1"] = map[string][]byte{
		"name":  []byte("John Doe"),
		"email": []byte("john@example.com"),
		"age":   []byte("30"),
	}
	hash.data["user:user2"] = map[string][]byte{
		"name":  []byte("Jane Doe"),
		"email": []byte("jane@example.com"),
		"age":   []byte("25"),
	}
	hash.data["user:user3"] = map[string][]byte{
		"name":  []byte("Bob Doe"),
		"email": []byte("bob@example.com"),
		"age":   []byte("40"),
	}

	results, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 unique results, got %d", len(results))
	}

	ids := map[string]bool{}
	for _, result := range results {
		ids[result.ID] = true
	}

	for _, id := range []string{"user1", "user2", "user3"} {
		if !ids[id] {
			t.Fatalf("Expected result for %s", id)
		}
	}
}

func TestHashRepository_Count_ScansAllPagesAndDeduplicates(t *testing.T) {
	ctx := context.Background()
	hash := newMockHash()
	kv := newMockKV()
	kv.scanPages = [][]string{
		{"user:user1", "user:user2"},
		{"user:user2", "user:user3"},
	}
	repo := repository.NewHashRepository[TestUser](hash, kv, "user")

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	if count != 3 {
		t.Fatalf("Expected count 3, got %d", count)
	}
}
