package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/DaiYuANg/arcgo/kvx/repository"
)

func TestHashRepository_SaveWithExpiration(t *testing.T) {
	ctx := context.Background()
	hash := newMockHash()
	kv := newMockKV()
	repo := repository.NewHashRepository[TestUser](hash, kv, "user")

	user := &TestUser{
		ID:    "user1",
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}

	err := repo.SaveWithExpiration(ctx, user, time.Hour)
	if err != nil {
		t.Fatalf("SaveWithExpiration failed: %v", err)
	}

	if _, exists := kv.expiration["user:user1"]; !exists {
		t.Errorf("Expiration not set")
	}
}

func TestHashRepository_UpdateField(t *testing.T) {
	ctx := context.Background()
	hash := newMockHash()
	kv := newMockKV()
	repo := repository.NewHashRepository[TestUser](hash, kv, "user")

	hash.data["user:user1"] = map[string][]byte{
		"name":  []byte("John Doe"),
		"email": []byte("john@example.com"),
		"age":   []byte("30"),
	}

	err := repo.UpdateField(ctx, "user1", "Name", "Jane Doe")
	if err != nil {
		t.Fatalf("UpdateField failed: %v", err)
	}
}

func TestHashRepository_IncrementField(t *testing.T) {
	ctx := context.Background()
	hash := newMockHash()
	kv := newMockKV()
	repo := repository.NewHashRepository[TestUser](hash, kv, "user")

	hash.data["user:user1"] = map[string][]byte{
		"name":  []byte("John Doe"),
		"email": []byte("john@example.com"),
		"age":   []byte("30"),
	}

	newVal, err := repo.IncrementField(ctx, "user1", "Age", 5)
	if err != nil {
		t.Fatalf("IncrementField failed: %v", err)
	}

	_ = newVal
}
