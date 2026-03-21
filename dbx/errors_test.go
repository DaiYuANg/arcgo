package dbx

import (
	"errors"
	"fmt"
	"testing"
)

func TestStructuredErrors_Is(t *testing.T) {
	t.Run("PrimaryKeyUnmappedError", func(t *testing.T) {
		err := &PrimaryKeyUnmappedError{Column: "id"}
		if !errors.Is(err, ErrPrimaryKeyUnmapped) {
			t.Error("errors.Is(err, ErrPrimaryKeyUnmapped) should be true")
		}
	})

	t.Run("UnknownCodecError", func(t *testing.T) {
		err := &UnknownCodecError{Name: "custom"}
		if !errors.Is(err, ErrUnknownCodec) {
			t.Error("errors.Is(err, ErrUnknownCodec) should be true")
		}
	})

	t.Run("UnmappedColumnError", func(t *testing.T) {
		err := &UnmappedColumnError{Column: "missing_col"}
		if !errors.Is(err, ErrUnmappedColumn) {
			t.Error("errors.Is(err, ErrUnmappedColumn) should be true")
		}
	})
}

func TestStructuredErrors_As(t *testing.T) {
	t.Run("PrimaryKeyUnmappedError", func(t *testing.T) {
		err := &PrimaryKeyUnmappedError{Column: "role_id"}
		var target *PrimaryKeyUnmappedError
		if !errors.As(err, &target) {
			t.Fatal("errors.As should succeed")
		}
		if target.Column != "role_id" {
			t.Errorf("expected Column=%q, got %q", "role_id", target.Column)
		}
	})

	t.Run("UnknownCodecError", func(t *testing.T) {
		err := &UnknownCodecError{Name: "jsonb"}
		var target *UnknownCodecError
		if !errors.As(err, &target) {
			t.Fatal("errors.As should succeed")
		}
		if target.Name != "jsonb" {
			t.Errorf("expected Name=%q, got %q", "jsonb", target.Name)
		}
	})

	t.Run("UnmappedColumnError", func(t *testing.T) {
		err := &UnmappedColumnError{Column: "deleted_at"}
		var target *UnmappedColumnError
		if !errors.As(err, &target) {
			t.Fatal("errors.As should succeed")
		}
		if target.Column != "deleted_at" {
			t.Errorf("expected Column=%q, got %q", "deleted_at", target.Column)
		}
	})
}

func TestStructuredErrors_Wrapped(t *testing.T) {
	// When structured error is wrapped by fmt.Errorf, As and Is should still work.
	err := &UnknownCodecError{Name: "csv"}
	wrapped := fmt.Errorf("mapper init: %w", err)

	if !errors.Is(wrapped, ErrUnknownCodec) {
		t.Error("errors.Is(wrapped, ErrUnknownCodec) should be true")
	}
	var target *UnknownCodecError
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As should succeed on wrapped error")
	}
	if target.Name != "csv" {
		t.Errorf("expected Name=%q, got %q", "csv", target.Name)
	}
}
