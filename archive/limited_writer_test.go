package archive

import (
	"bytes"
	"crypto/rand"
	"io"
	"os"
	"syscall"
	"testing"
)

type recordingWriter struct {
	buf      bytes.Buffer
	sizes    []int
	failIfGT int
	failErr  error
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	w.sizes = append(w.sizes, len(p))
	if w.failIfGT > 0 && len(p) > w.failIfGT {
		return 0, w.failErr
	}
	return w.buf.Write(p)
}

func TestLimitedWriterSplitsLargeWrites(t *testing.T) {
	inner := &recordingWriter{}
	w := newLimitedWriterN(inner, 4)
	input := []byte("abcdefghij")
	n, err := w.Write(input)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(input) {
		t.Fatalf("wrote %d bytes, want %d", n, len(input))
	}
	if !bytes.Equal(inner.buf.Bytes(), input) {
		t.Fatalf("written data = %q, want %q", inner.buf.Bytes(), input)
	}
	wantSizes := []int{4, 4, 2}
	if len(inner.sizes) != len(wantSizes) {
		t.Fatalf("write sizes = %v, want %v", inner.sizes, wantSizes)
	}
	for i, size := range wantSizes {
		if inner.sizes[i] != size {
			t.Fatalf("write sizes = %v, want %v", inner.sizes, wantSizes)
		}
	}
}

func TestLimitedWriterRetriesWindowsNoSystemResources(t *testing.T) {
	oldDelay := noSystemResourcesDelay
	noSystemResourcesDelay = 0
	t.Cleanup(func() { noSystemResourcesDelay = oldDelay })

	failErr := &os.PathError{
		Op:   "write",
		Path: `D:\xxxxxxxxxxxxxxxx.bks`,
		Err:  windowsErrorNoSystemResources,
	}
	inner := &recordingWriter{
		failIfGT: 64 * 1024,
		failErr:  failErr,
	}
	w := newLimitedWriterN(inner, 256*1024)
	input := bytes.Repeat([]byte("x"), 200*1024)
	n, err := w.Write(input)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(input) {
		t.Fatalf("wrote %d bytes, want %d", n, len(input))
	}
	if !bytes.Equal(inner.buf.Bytes(), input) {
		t.Fatal("written data mismatch after retry")
	}
	if len(inner.sizes) < 2 {
		t.Fatalf("expected a failed large write followed by smaller writes, got %v", inner.sizes)
	}
	if inner.sizes[0] != 200*1024 {
		t.Fatalf("first write size = %d, want %d", inner.sizes[0], 200*1024)
	}
	for _, size := range inner.sizes[1:] {
		if size > 64*1024 {
			t.Fatalf("retry write size %d exceeds 64KiB", size)
		}
	}
}

func TestLimitedWriterPropagatesNonResourceErrors(t *testing.T) {
	inner := &recordingWriter{
		failIfGT: 1,
		failErr:  os.ErrClosed,
	}
	w := newLimitedWriterN(inner, 8)
	_, err := w.Write([]byte("abcd"))
	if err != os.ErrClosed {
		t.Fatalf("error = %v, want %v", err, os.ErrClosed)
	}
}

func TestExportRetriesWindowsNoSystemResources(t *testing.T) {
	oldDelay := noSystemResourcesDelay
	noSystemResourcesDelay = 0
	t.Cleanup(func() { noSystemResourcesDelay = oldDelay })

	input := make([]byte, 128*1024)
	if _, err := rand.Read(input); err != nil {
		t.Fatal(err)
	}

	inner := &recordingWriter{
		failIfGT: 64 * 1024,
		failErr: &os.PathError{
			Op:   "write",
			Path: `D:\xxxxxxxxxxxxxxxx.bks`,
			Err:  windowsErrorNoSystemResources,
		},
	}
	bks := BksArchive{
		Password:  testPassword,
		ChunkSize: 128 * 1024,
	}
	if err := bks.Export("file.bin", bytes.NewReader(input), inner); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	var output bytes.Buffer
	if err := bks.Import(bytes.NewReader(inner.buf.Bytes()), func(name string) (io.Writer, error) {
		if name != "file.bin" {
			t.Fatalf("name = %q, want file.bin", name)
		}
		return &output, nil
	}); err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if !bytes.Equal(output.Bytes(), input) {
		t.Fatal("roundtrip content mismatch after resource-error retries")
	}
}

func TestIsWindowsNoSystemResources(t *testing.T) {
	pathErr := &os.PathError{
		Op:   "write",
		Path: `D:\xxxxxxxxxxxxxxxx.bks`,
		Err:  syscall.Errno(1450),
	}
	if !isWindowsNoSystemResources(pathErr) {
		t.Fatal("expected PathError wrapping errno 1450 to match")
	}
	if isWindowsNoSystemResources(io.ErrUnexpectedEOF) {
		t.Fatal("did not expect a generic IO error to match")
	}
}
