package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceFileOverwritesExisting(t *testing.T) {
	tmp := t.TempDir()
	oldPath := filepath.Join(tmp, "old.txt")
	newPath := filepath.Join(tmp, "new.txt")

	if err := os.WriteFile(oldPath, []byte("replacement"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte("previous"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile(oldPath, newPath); err != nil {
		t.Fatalf("replaceFile failed: %v", err)
	}

	got, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "replacement" {
		t.Fatalf("replaced content = %q, want replacement", got)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("source file still exists after replace: %v", err)
	}
}
