package core

import (
	"os"
	"path/filepath"
	"testing"

	"bakashier/archive"
)

func writeMovedDirCandidate(t *testing.T, srcDir string, name string, files ...string) {
	t.Helper()
	dir := filepath.Join(srcDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(file), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func writePreviousDirEntries(t *testing.T, distDir string, hideName string, files ...string) {
	t.Helper()
	dir := filepath.Join(distDir, hideName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	entries := make([]archive.DirectoryEntry, 0, len(files))
	for index, file := range files {
		entries = append(entries, archive.DirectoryEntry{
			Type:     archive.File,
			RealName: file,
			HideName: string(rune('a' + index)),
		})
	}
	if err := saveDirectoryEntries(filepath.Join(dir, "_directory_.bks"), "/old", entries, testPassword, 1024); err != nil {
		t.Fatal(err)
	}
}

func TestCheckMovedDirsChoosesHighestContentSimilarity(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	distDir := filepath.Join(tmp, "dist")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(distDir, 0755); err != nil {
		t.Fatal(err)
	}

	writeMovedDirCandidate(t, srcDir, "renamed", "a", "b", "c", "d")
	writePreviousDirEntries(t, distDir, "most-similar", "a", "b", "c", "d")
	writePreviousDirEntries(t, distDir, "least-similar", "a")

	files, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatal(err)
	}
	entries := []archive.DirectoryEntry{
		{Type: archive.Directory, RealName: "old-most", HideName: "most-similar"},
		{Type: archive.Directory, RealName: "old-least", HideName: "least-similar"},
	}

	moved, err := checkMovedDirs(srcDir, distDir, entries, files, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if len(moved) != 1 {
		t.Fatalf("moved count = %d, want 1: %+v", len(moved), moved)
	}
	if moved[0].HideName != "most-similar" {
		t.Fatalf("selected hidden directory = %q, want %q", moved[0].HideName, "most-similar")
	}
}

func TestCheckMovedDirsDoesNotReusePreviousDirectory(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	distDir := filepath.Join(tmp, "dist")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(distDir, 0755); err != nil {
		t.Fatal(err)
	}

	writeMovedDirCandidate(t, srcDir, "renamed-one", "same")
	writeMovedDirCandidate(t, srcDir, "renamed-two", "same")
	writePreviousDirEntries(t, distDir, "old-hidden", "same")

	files, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatal(err)
	}
	entries := []archive.DirectoryEntry{
		{Type: archive.Directory, RealName: "old", HideName: "old-hidden"},
	}

	moved, err := checkMovedDirs(srcDir, distDir, entries, files, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if len(moved) != 1 {
		t.Fatalf("one previous directory was matched %d times: %+v", len(moved), moved)
	}
}
