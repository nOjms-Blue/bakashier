package archive

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

const testPassword = "test-password"

type errWriter struct {
	err error
}

func (w errWriter) Write(p []byte) (int, error) {
	return 0, w.err
}

func TestBksArchive(t *testing.T) {
	bks := BksArchive{
		Password:  "password",
		ChunkSize: 1024,
	}

	name := "example.bin"
	input := bytes.Repeat([]byte("0123456789abcdef"), 640+3)

	var archived bytes.Buffer
	err := bks.Export(name, bytes.NewReader(input), &archived)
	if err != nil {
		t.Fatalf("bks.Export() failed: %v", err)
	}

	var output bytes.Buffer
	getWriterCalled := 0
	receivedName := ""
	err = bks.Import(
		bytes.NewReader(archived.Bytes()),
		func(name string) (io.Writer, error) {
			getWriterCalled++
			receivedName = name
			return &output, nil
		},
	)
	if err != nil {
		t.Fatalf("bks.Import() failed: %v", err)
	}

	if getWriterCalled != 1 {
		t.Errorf("getWriter call count = %d, want 1", getWriterCalled)
	}
	if receivedName != name {
		t.Errorf("getWriter name = %q, want %q", receivedName, name)
	}

	if !bytes.Equal(output.Bytes(), input) {
		t.Errorf("input data and output data did not match")
	}
}

func TestBksArchiveEmptyFile(t *testing.T) {
	bks := BksArchive{
		Password:  testPassword,
		ChunkSize: 1024,
	}

	var archived bytes.Buffer
	if err := bks.Export("empty.txt", bytes.NewReader(nil), &archived); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	var output bytes.Buffer
	err := bks.Import(bytes.NewReader(archived.Bytes()), func(name string) (io.Writer, error) {
		if name != "empty.txt" {
			t.Errorf("name = %q, want empty.txt", name)
		}
		return &output, nil
	})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("restored empty file has size %d", output.Len())
	}
}

func TestBksArchiveWrongPassword(t *testing.T) {
	exporter := BksArchive{
		Password:  testPassword,
		ChunkSize: 1024,
	}
	var archived bytes.Buffer
	if err := exporter.Export("secret.txt", bytes.NewReader([]byte("secret data")), &archived); err != nil {
		t.Fatal(err)
	}

	importer := BksArchive{Password: "wrong-password"}
	err := importer.Import(bytes.NewReader(archived.Bytes()), func(name string) (io.Writer, error) {
		return io.Discard, nil
	})
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

func TestBksArchiveTruncated(t *testing.T) {
	bks := BksArchive{
		Password:  testPassword,
		ChunkSize: 1024,
	}
	var archived bytes.Buffer
	if err := bks.Export("source.bin", bytes.NewReader(bytes.Repeat([]byte("x"), 4096)), &archived); err != nil {
		t.Fatal(err)
	}
	full := archived.Bytes()

	for _, size := range []int{0, 1, 5, 9, 20, len(full) / 2, len(full) - 2} {
		err := bks.Import(bytes.NewReader(full[:size]), func(name string) (io.Writer, error) {
			return io.Discard, nil
		})
		if err == nil {
			t.Fatalf("expected error for archive truncated to %d bytes, got nil", size)
		}
	}
}

func TestBksArchiveOversizedChunkLength(t *testing.T) {
	bks := BksArchive{
		Password:  testPassword,
		ChunkSize: 1024,
	}
	var archived bytes.Buffer
	if err := bks.Export("source.bin", bytes.NewReader([]byte("data")), &archived); err != nil {
		t.Fatal(err)
	}
	full := archived.Bytes()

	nameLen := binary.BigEndian.Uint32(full[5:9])
	chunkLenOffset := 9 + int(nameLen) + 4
	tampered := append([]byte{}, full...)
	binary.BigEndian.PutUint64(tampered[chunkLenOffset:chunkLenOffset+8], ^uint64(0))

	err := bks.Import(bytes.NewReader(tampered), func(name string) (io.Writer, error) {
		return io.Discard, nil
	})
	if err == nil || !strings.Contains(err.Error(), "chunk too large") {
		t.Fatalf("expected 'chunk too large' error, got: %v", err)
	}
}

func TestBksArchiveRejectsOversizedDecompressedName(t *testing.T) {
	bks := BksArchive{
		Password:  testPassword,
		ChunkSize: 1024,
	}
	name := strings.Repeat("a", int(maxEncryptedNameSize)+1)

	var archived bytes.Buffer
	if err := bks.Export(name, bytes.NewReader(nil), &archived); err != nil {
		t.Fatal(err)
	}
	called := false
	err := bks.Import(bytes.NewReader(archived.Bytes()), func(name string) (io.Writer, error) {
		called = true
		return io.Discard, nil
	})
	if err == nil {
		t.Fatal("expected oversized decompressed name to be rejected")
	}
	if called {
		t.Fatal("getWriter was called with an oversized name")
	}
}

func TestBksArchiveTruncatedMaximumLengthDoesNotPreallocateClaimedSize(t *testing.T) {
	bks := BksArchive{
		Password:  testPassword,
		ChunkSize: 1024,
	}
	var archived bytes.Buffer
	if err := bks.Export("source.bin", bytes.NewReader([]byte("data")), &archived); err != nil {
		t.Fatal(err)
	}
	tampered := append([]byte{}, archived.Bytes()...)
	nameLen := binary.BigEndian.Uint32(tampered[5:9])
	chunkLenOffset := 9 + int(nameLen) + 4
	binary.BigEndian.PutUint64(tampered[chunkLenOffset:chunkLenOffset+8], maxEncryptedChunkSize)

	err := bks.Import(bytes.NewReader(tampered), func(name string) (io.Writer, error) {
		return io.Discard, nil
	})
	if err == nil || !strings.Contains(err.Error(), "truncated block") {
		t.Fatalf("expected truncated block error, got: %v", err)
	}
}

func TestBksArchiveCRCFailureDoesNotWriteInvalidChunk(t *testing.T) {
	bks := BksArchive{
		Password:  testPassword,
		ChunkSize: 1024,
	}
	var archived bytes.Buffer
	if err := bks.Export("source.bin", bytes.NewReader([]byte("data")), &archived); err != nil {
		t.Fatal(err)
	}
	tampered := append([]byte{}, archived.Bytes()...)
	tampered[len(tampered)-1] ^= 0xff

	var output bytes.Buffer
	err := bks.Import(bytes.NewReader(tampered), func(name string) (io.Writer, error) {
		return &output, nil
	})
	if err == nil {
		t.Fatal("expected CRC error")
	}
	if output.Len() != 0 {
		t.Fatalf("wrote %d bytes from a chunk that failed integrity validation", output.Len())
	}
}

func TestBksArchiveImportAllowsPathName(t *testing.T) {
	bks := BksArchive{
		Password:  testPassword,
		ChunkSize: 1024,
	}
	name := "/tmp/src/dir"
	content := []byte("directory entries")

	var archived bytes.Buffer
	if err := bks.Export(name, bytes.NewReader(content), &archived); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	receivedName := ""
	err := bks.Import(bytes.NewReader(archived.Bytes()), func(imported string) (io.Writer, error) {
		receivedName = imported
		return &output, nil
	})
	if err != nil {
		t.Fatalf("Import should accept path names used by _directory_.bks: %v", err)
	}
	if receivedName != name {
		t.Errorf("name = %q, want %q", receivedName, name)
	}
	if !bytes.Equal(output.Bytes(), content) {
		t.Fatal("content mismatch")
	}
}

func TestExportPropagatesWriterError(t *testing.T) {
	writeErr := os.ErrClosed
	bks := BksArchive{
		Password:  testPassword,
		ChunkSize: 1024,
	}
	err := bks.Export("name", bytes.NewReader(nil), errWriter{err: writeErr})
	if err == nil {
		t.Fatal("expected writer error to be propagated, got nil")
	}
}

func TestExportRejectsInvalidChunkSize(t *testing.T) {
	for _, size := range []uint64{0, MAX_CHUNK_SIZE + 1} {
		bks := BksArchive{
			Password:  testPassword,
			ChunkSize: size,
		}
		err := bks.Export("name", bytes.NewReader(nil), io.Discard)
		if err == nil {
			t.Fatalf("expected error for chunk size %d, got nil", size)
		}
	}
}

func TestIsSafeFileName(t *testing.T) {
	safe := []string{"file.txt", "ファイル.dat", ".hidden", "a b c"}
	unsafe := []string{"", ".", "..", "a/b", `a\b`, "../x", "/etc/passwd", "a\x00b"}

	for _, name := range safe {
		if !IsSafeFileName(name) {
			t.Errorf("expected %q to be safe", name)
		}
	}
	for _, name := range unsafe {
		if IsSafeFileName(name) {
			t.Errorf("expected %q to be unsafe", name)
		}
	}
}

func TestDirectoryEntriesRoundtrip(t *testing.T) {
	modTime := time.Unix(1_700_000_000, 123)
	want := []DirectoryEntry{
		{
			Type:     Directory,
			RealName: "src",
			HideName: "abc123",
			Size:     0,
			ModTime:  modTime,
		},
		{
			Type:     File,
			RealName: "日本語.txt",
			HideName: "def456",
			Size:     42,
			ModTime:  modTime,
		},
	}

	index := 0
	var buf bytes.Buffer
	err := ExportDirectoryEntries(func(entry *DirectoryEntry) error {
		if index >= len(want) {
			return io.EOF
		}
		*entry = want[index]
		index++
		return nil
	}, &buf)
	if err != nil {
		t.Fatalf("ExportDirectoryEntries failed: %v", err)
	}

	var got []DirectoryEntry
	err = ImportDirectoryEntries(bytes.NewReader(buf.Bytes()), func(entry DirectoryEntry) error {
		got = append(got, entry)
		return nil
	})
	if err != nil {
		t.Fatalf("ImportDirectoryEntries failed: %v", err)
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

func TestDirectoryEntriesEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := ExportDirectoryEntries(func(entry *DirectoryEntry) error {
		return io.EOF
	}, &buf)
	if err != nil {
		t.Fatalf("ExportDirectoryEntries failed: %v", err)
	}

	called := false
	err = ImportDirectoryEntries(bytes.NewReader(buf.Bytes()), func(entry DirectoryEntry) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("ImportDirectoryEntries failed: %v", err)
	}
	if called {
		t.Fatal("expected no entries")
	}
}

func TestExportDirectoryEntriesRejectsLongName(t *testing.T) {
	longName := strings.Repeat("a", MAX_NAME_SIZE+1)
	called := false
	err := ExportDirectoryEntries(func(entry *DirectoryEntry) error {
		if called {
			return io.EOF
		}
		called = true
		*entry = DirectoryEntry{
			Type:     File,
			RealName: longName,
			HideName: "hide",
		}
		return nil
	}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "real name length is out of range") {
		t.Fatalf("expected real name length error, got: %v", err)
	}
}

func TestImportDirectoryEntriesRejectsLongName(t *testing.T) {
	payload := make([]byte, 1+4+4)
	payload[0] = byte(File)
	binary.BigEndian.PutUint32(payload[1:5], MAX_NAME_SIZE+1)
	binary.BigEndian.PutUint32(payload[5:9], 1)

	err := ImportDirectoryEntries(bytes.NewReader(payload), func(entry DirectoryEntry) error {
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "real name length is out of range") {
		t.Fatalf("expected real name length error, got: %v", err)
	}
}
