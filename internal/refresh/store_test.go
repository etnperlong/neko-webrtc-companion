package refresh

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
)

func TestFileStore_ReadAndWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	store := NewFileStore(path, 0o600)
	ctx := context.Background()

	initial := []byte("initial data")
	if err := store.Write(ctx, initial); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	read, err := store.Read(ctx)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if !bytes.Equal(read, initial) {
		t.Fatalf("expected %q, got %q", initial, read)
	}

	updated := []byte("updated data")
	if err := store.Write(ctx, updated); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	read, err = store.Read(ctx)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if !bytes.Equal(read, updated) {
		t.Fatalf("expected %q, got %q", updated, read)
	}
}
