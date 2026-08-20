package archive

import (
	"bytes"
	"io"
	"testing"
)

func TestBksArchive(t *testing.T) {
	bks := BksArchive{
		Password:  "password",
		ChunkSize: 1024,
	}

	name := "example.bin"
	input := bytes.Repeat([]byte("0123456789abcdef"), 640+3)

	// エクスポート
	var archive bytes.Buffer
	err := bks.Export(name, bytes.NewReader(input), &archive)
	if err != nil {
		t.Fatalf("bks.Export() failed: %v", err)
	}

	// インポート
	var output bytes.Buffer
	getWriterCalled := 0
	receivedName := ""
	err = bks.Import(
		bytes.NewReader(archive.Bytes()),
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
