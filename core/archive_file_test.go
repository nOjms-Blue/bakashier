package core

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bakashier/archive"
)

const testPassword = "test-password"

func TestArchiveFileRoundtrip(t *testing.T) {
	tmp := t.TempDir()

	content := bytes.Repeat([]byte("0123456789abcdef"), 640+3)
	srcFile := filepath.Join(tmp, "source.bin")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	archiveFile := filepath.Join(tmp, "archive.bks")
	if err := exportArchiveFile(srcFile, archiveFile, "source.bin", testPassword, 1024); err != nil {
		t.Fatalf("exportArchiveFile failed: %v", err)
	}

	restoreDir := filepath.Join(tmp, "restore")
	if err := os.MkdirAll(restoreDir, 0755); err != nil {
		t.Fatal(err)
	}
	restored, err := importArchiveFile(archiveFile, restoreDir, testPassword)
	if err != nil {
		t.Fatalf("importArchiveFile failed: %v", err)
	}
	if restored != filepath.Join(restoreDir, "source.bin") {
		t.Fatalf("unexpected restored path: %s", restored)
	}

	restoredContent, err := os.ReadFile(restored)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, restoredContent) {
		t.Fatalf("restored content mismatch: want %d bytes, got %d bytes", len(content), len(restoredContent))
	}
}

func TestArchiveFileEmptyFile(t *testing.T) {
	tmp := t.TempDir()

	srcFile := filepath.Join(tmp, "empty.txt")
	if err := os.WriteFile(srcFile, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	archiveFile := filepath.Join(tmp, "empty.bks")
	if err := exportArchiveFile(srcFile, archiveFile, "empty.txt", testPassword, 1024); err != nil {
		t.Fatal(err)
	}

	restoreDir := filepath.Join(tmp, "restore")
	if err := os.MkdirAll(restoreDir, 0755); err != nil {
		t.Fatal(err)
	}
	restored, err := importArchiveFile(archiveFile, restoreDir, testPassword)
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(restored)
	if err != nil {
		t.Fatalf("restored empty file does not exist: %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("restored empty file has size %d", info.Size())
	}
}

func TestArchiveFileWrongPassword(t *testing.T) {
	tmp := t.TempDir()

	srcFile := filepath.Join(tmp, "secret.txt")
	if err := os.WriteFile(srcFile, []byte("secret data"), 0644); err != nil {
		t.Fatal(err)
	}

	archiveFile := filepath.Join(tmp, "secret.bks")
	if err := exportArchiveFile(srcFile, archiveFile, "secret.txt", testPassword, 1024); err != nil {
		t.Fatal(err)
	}

	restoreDir := filepath.Join(tmp, "restore")
	if err := os.MkdirAll(restoreDir, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := importArchiveFile(archiveFile, restoreDir, "wrong-password"); err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

func TestImportArchiveFileTruncated(t *testing.T) {
	tmp := t.TempDir()

	srcFile := filepath.Join(tmp, "source.bin")
	if err := os.WriteFile(srcFile, bytes.Repeat([]byte("x"), 4096), 0644); err != nil {
		t.Fatal(err)
	}
	archiveFile := filepath.Join(tmp, "archive.bks")
	if err := exportArchiveFile(srcFile, archiveFile, "source.bin", testPassword, 1024); err != nil {
		t.Fatal(err)
	}

	full, err := os.ReadFile(archiveFile)
	if err != nil {
		t.Fatal(err)
	}

	restoreDir := filepath.Join(tmp, "restore")
	if err := os.MkdirAll(restoreDir, 0755); err != nil {
		t.Fatal(err)
	}

	for _, size := range []int{0, 1, 5, 9, 20, len(full) / 2, len(full) - 2} {
		truncated := filepath.Join(tmp, "truncated.bks")
		if err := os.WriteFile(truncated, full[:size], 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := importArchiveFile(truncated, restoreDir, testPassword); err == nil {
			t.Fatalf("expected error for archive truncated to %d bytes, got nil", size)
		}
	}
}

func TestImportArchiveFileOversizedChunkLength(t *testing.T) {
	tmp := t.TempDir()

	srcFile := filepath.Join(tmp, "source.bin")
	if err := os.WriteFile(srcFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	archiveFile := filepath.Join(tmp, "archive.bks")
	if err := exportArchiveFile(srcFile, archiveFile, "source.bin", testPassword, 1024); err != nil {
		t.Fatal(err)
	}

	full, err := os.ReadFile(archiveFile)
	if err != nil {
		t.Fatal(err)
	}

	nameLen := binary.BigEndian.Uint32(full[5:9])
	chunkLenOffset := 9 + int(nameLen) + 4
	binary.BigEndian.PutUint64(full[chunkLenOffset:chunkLenOffset+8], ^uint64(0))

	tampered := filepath.Join(tmp, "tampered.bks")
	if err := os.WriteFile(tampered, full, 0644); err != nil {
		t.Fatal(err)
	}

	restoreDir := filepath.Join(tmp, "restore")
	if err := os.MkdirAll(restoreDir, 0755); err != nil {
		t.Fatal(err)
	}
	_, err = importArchiveFile(tampered, restoreDir, testPassword)
	if err == nil || !strings.Contains(err.Error(), "chunk too large") {
		t.Fatalf("expected 'chunk too large' error, got: %v", err)
	}
}

func TestImportArchiveFileRejectsUnsafeFileName(t *testing.T) {
	tmp := t.TempDir()

	for _, name := range []string{"../evil.txt", "..", "a/b.txt", `..\evil.txt`, ""} {
		archiveFile := filepath.Join(tmp, "unsafe.bks")
		fp, err := os.Create(archiveFile)
		if err != nil {
			t.Fatal(err)
		}
		bks := archive.BksArchive{
			Password:  testPassword,
			ChunkSize: 1024,
		}
		err = bks.Export(name, bytes.NewReader([]byte("malicious")), fp)
		fp.Close()
		if err != nil {
			t.Fatal(err)
		}

		restoreDir := filepath.Join(tmp, "restore")
		if err := os.MkdirAll(restoreDir, 0755); err != nil {
			t.Fatal(err)
		}
		if _, err := importArchiveFile(archiveFile, restoreDir, testPassword); err == nil {
			t.Fatalf("expected error for unsafe name %q, got nil", name)
		}
		if _, err := os.Stat(filepath.Join(tmp, "evil.txt")); err == nil {
			t.Fatalf("file escaped restore directory for name %q", name)
		}
	}
}

func TestExportArchiveFileFailurePreservesExistingDestination(t *testing.T) {
	tmp := t.TempDir()
	srcFile := filepath.Join(tmp, "source.txt")
	dstFile := filepath.Join(tmp, "archive.bks")
	original := []byte("previous valid archive")

	if err := os.WriteFile(srcFile, []byte("new data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dstFile, original, 0644); err != nil {
		t.Fatal(err)
	}

	if err := exportArchiveFile(srcFile, dstFile, "source.txt", testPassword, 0); err == nil {
		t.Fatal("expected export failure for invalid chunk size")
	}

	got, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("existing destination was removed: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("existing destination changed after failed export: got %q", got)
	}
}

func TestImportArchiveFileFailurePreservesExistingDestination(t *testing.T) {
	tmp := t.TempDir()
	srcFile := filepath.Join(tmp, "source.txt")
	archiveFile := filepath.Join(tmp, "archive.bks")
	restoreDir := filepath.Join(tmp, "restore")
	existingFile := filepath.Join(restoreDir, "source.txt")
	original := []byte("existing restored data")

	if err := os.WriteFile(srcFile, []byte("new data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exportArchiveFile(srcFile, archiveFile, "source.txt", testPassword, 1024); err != nil {
		t.Fatal(err)
	}
	archiveData, err := os.ReadFile(archiveFile)
	if err != nil {
		t.Fatal(err)
	}
	archiveData[len(archiveData)-1] ^= 0xff
	if err := os.WriteFile(archiveFile, archiveData, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(restoreDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existingFile, original, 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := importArchiveFile(archiveFile, restoreDir, testPassword); err == nil {
		t.Fatal("expected import failure for corrupted archive")
	}

	got, err := os.ReadFile(existingFile)
	if err != nil {
		t.Fatalf("existing destination was removed: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("existing destination changed after failed import: got %q", got)
	}
}

func TestImportArchiveFileDoesNotFollowDestinationSymlink(t *testing.T) {
	tmp := t.TempDir()
	srcFile := filepath.Join(tmp, "source.txt")
	archiveFile := filepath.Join(tmp, "archive.bks")
	restoreDir := filepath.Join(tmp, "restore")
	outsideFile := filepath.Join(tmp, "outside.txt")

	if err := os.WriteFile(srcFile, []byte("restored data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exportArchiveFile(srcFile, archiveFile, "source.txt", testPassword, 1024); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(restoreDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outsideFile, []byte("outside data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideFile, filepath.Join(restoreDir, "source.txt")); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}

	if _, err := importArchiveFile(archiveFile, restoreDir, testPassword); err != nil {
		t.Fatalf("importArchiveFile failed: %v", err)
	}

	outside, err := os.ReadFile(outsideFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(outside) != "outside data" {
		t.Fatalf("restore followed destination symlink and changed outside file: got %q", outside)
	}
	restored, err := os.ReadFile(filepath.Join(restoreDir, "source.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != "restored data" {
		t.Fatalf("restored content = %q, want %q", restored, "restored data")
	}
}

func TestSaveLoadDirectoryEntries(t *testing.T) {
	tmp := t.TempDir()

	modTime := time.Unix(1_700_000_000, 123)
	want := []archive.DirectoryEntry{
		{
			Type:     archive.Directory,
			RealName: "src",
			HideName: "abc123",
			Size:     0,
			ModTime:  modTime,
		},
		{
			Type:     archive.File,
			RealName: "file.txt",
			HideName: "def456",
			Size:     99,
			ModTime:  modTime,
		},
	}

	directoryEntryFile := filepath.Join(tmp, "_directory_.bks")
	if err := saveDirectoryEntries(directoryEntryFile, "/tmp/src/dir", want, testPassword, 1024); err != nil {
		t.Fatalf("saveDirectoryEntries failed: %v", err)
	}

	got, err := loadDirectoryEntries(directoryEntryFile, testPassword)
	if err != nil {
		t.Fatalf("loadDirectoryEntries failed: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("entry count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Type != want[i].Type ||
			got[i].RealName != want[i].RealName ||
			got[i].HideName != want[i].HideName ||
			got[i].Size != want[i].Size ||
			!got[i].ModTime.Equal(want[i].ModTime) {
			t.Errorf("entry[%d] mismatch: got %+v want %+v", i, got[i], want[i])
		}
	}
}

func TestSaveDirectoryEntriesFailurePreservesExistingDestination(t *testing.T) {
	tmp := t.TempDir()
	directoryEntryFile := filepath.Join(tmp, "_directory_.bks")
	original := []byte("previous valid directory archive")

	if err := os.WriteFile(directoryEntryFile, original, 0644); err != nil {
		t.Fatal(err)
	}
	err := saveDirectoryEntries(
		directoryEntryFile,
		"/source",
		[]archive.DirectoryEntry{{Type: archive.File, RealName: "file.txt", HideName: "hidden"}},
		testPassword,
		0,
	)
	if err == nil {
		t.Fatal("expected save failure for invalid chunk size")
	}

	got, readErr := os.ReadFile(directoryEntryFile)
	if readErr != nil {
		t.Fatalf("existing directory archive was removed: %v", readErr)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("existing directory archive changed after failed save: got %q", got)
	}
}

func TestLoadDirectoryEntriesMissingFile(t *testing.T) {
	entries, err := loadDirectoryEntries(filepath.Join(t.TempDir(), "_directory_.bks"), testPassword)
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty entries, got %d", len(entries))
	}
}

func TestLoadDirectoryEntriesPropagatesNonExistenceRelatedStatErrors(t *testing.T) {
	tmp := t.TempDir()
	notDirectory := filepath.Join(tmp, "not-a-directory")
	if err := os.WriteFile(notDirectory, []byte("file"), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := loadDirectoryEntries(filepath.Join(notDirectory, "_directory_.bks"), testPassword)
	if err == nil {
		t.Fatalf("expected ENOTDIR to be returned, got entries: %+v", entries)
	}
}
