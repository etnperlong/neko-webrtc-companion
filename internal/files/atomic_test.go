package files

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestWriteFileAtomic_ReplacesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.txt")

	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatalf("os.WriteFile initial: %v", err)
	}

	if err := WriteFileAtomic(path, []byte("new"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile: %v", err)
	}
	if string(got) != "new" {
		t.Fatalf("unexpected content: %q", got)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("unexpected perms: %o", info.Mode().Perm())
	}
}

func TestWriteFileAtomic_PreservesInodeForExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.txt")

	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatalf("os.WriteFile initial: %v", err)
	}

	before, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat before: %v", err)
	}
	beforeStat, ok := before.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatal("expected syscall.Stat_t for initial file")
	}

	if err := WriteFileAtomic(path, []byte("new"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	after, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat after: %v", err)
	}
	afterStat, ok := after.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatal("expected syscall.Stat_t for updated file")
	}

	if beforeStat.Ino != afterStat.Ino {
		t.Fatalf("expected inode to stay the same, got %d -> %d", beforeStat.Ino, afterStat.Ino)
	}
}
