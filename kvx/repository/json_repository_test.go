package repository_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/DaiYuANg/arcgo/kvx/repository"
)

func TestJSONRepository_ExistsBatch_UsesKVExists(t *testing.T) {
	ctx := context.Background()
	client := newMockJSON()
	kv := newMockKV()
	kv.data["user:user1"] = []byte("exists")
	kv.data["user:user3"] = []byte("exists")

	repo := repository.NewJSONRepository[TestUser](client, kv, "user")

	results, err := repo.ExistsBatch(ctx, []string{"user1", "user2", "user3"})
	if err != nil {
		t.Fatalf("ExistsBatch failed: %v", err)
	}

	expected := map[string]bool{
		"user1": true,
		"user2": false,
		"user3": true,
	}

	for id, want := range expected {
		if results[id] != want {
			t.Fatalf("expected %s existence to be %v, got %v", id, want, results[id])
		}
	}
}

func TestJSONRepository_FindAll_ScansAllPagesAndDeduplicates(t *testing.T) {
	ctx := context.Background()
	client := newMockJSON()
	kv := newMockKV()
	kv.scanPages = [][]string{
		{"user:user1", "user:user2"},
		{"user:user2", "user:user3"},
	}
	repo := repository.NewJSONRepository[TestUser](client, kv, "user")

	for _, user := range []*TestUser{
		{ID: "user1", Name: "John Doe", Email: "john@example.com", Age: 30},
		{ID: "user2", Name: "Jane Doe", Email: "jane@example.com", Age: 25},
		{ID: "user3", Name: "Bob Doe", Email: "bob@example.com", Age: 40},
	} {
		payload, err := json.Marshal(user)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}
		client.data["user:"+user.ID] = payload
	}

	results, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	if results.Len() != 3 {
		t.Fatalf("Expected 3 unique results, got %d", results.Len())
	}

	ids := map[string]bool{}
	results.Range(func(_ int, result *TestUser) bool {
		ids[result.ID] = true
		return true
	})

	for _, id := range []string{"user1", "user2", "user3"} {
		if !ids[id] {
			t.Fatalf("Expected result for %s", id)
		}
	}
}

func TestJSONRepository_Count_ScansAllPagesAndDeduplicates(t *testing.T) {
	ctx := context.Background()
	client := newMockJSON()
	kv := newMockKV()
	kv.scanPages = [][]string{
		{"user:user1", "user:user2"},
		{"user:user2", "user:user3"},
	}
	repo := repository.NewJSONRepository[TestUser](client, kv, "user")

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	if count != 3 {
		t.Fatalf("Expected count 3, got %d", count)
	}
}

func TestJSONRepository_Save_ReplacesStaleIndexes(t *testing.T) {
	ctx := context.Background()
	client := newMockJSON()
	kv := newMockKV()
	repo := repository.NewJSONRepository[TestUser](client, kv, "user")

	original := &TestUser{ID: "user1", Name: "John Doe", Email: "old@example.com", Age: 30}
	updated := &TestUser{ID: "user1", Name: "John Doe", Email: "new@example.com", Age: 31}

	payload, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	client.data["user:user1"] = payload
	kv.data["user:idx:email:old@example.com:user1"] = []byte("1")
	kv.data["user:idx:age:30:user1"] = []byte("1")

	if err := repo.Save(ctx, updated); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if _, ok := kv.data["user:idx:email:old@example.com:user1"]; ok {
		t.Fatalf("stale email index should be removed")
	}
	if _, ok := kv.data["user:idx:age:30:user1"]; ok {
		t.Fatalf("stale age index should be removed")
	}
	if _, ok := kv.data["user:idx:email:new@example.com:user1"]; !ok {
		t.Fatalf("new email index should exist")
	}
	if _, ok := kv.data["user:idx:age:31:user1"]; !ok {
		t.Fatalf("new age index should exist")
	}
}

func TestJSONRepository_UpdateField_ReplacesIndexedFieldEntry(t *testing.T) {
	ctx := context.Background()
	client := newMockJSON()
	kv := newMockKV()
	repo := repository.NewJSONRepository[TestUser](client, kv, "user")

	user := &TestUser{ID: "user1", Name: "John Doe", Email: "old@example.com", Age: 30}
	payload, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	client.data["user:user1"] = payload
	kv.data["user:idx:email:old@example.com:user1"] = []byte("1")

	if err := repo.UpdateField(ctx, "user1", "$.email", "new@example.com"); err != nil {
		t.Fatalf("UpdateField failed: %v", err)
	}

	if _, ok := kv.data["user:idx:email:old@example.com:user1"]; ok {
		t.Fatalf("old email index should be removed")
	}
	if _, ok := kv.data["user:idx:email:new@example.com:user1"]; !ok {
		t.Fatalf("new email index should exist")
	}
}
