package utils

import (
	"bytes"
	"errors"
	"testing"
)

func TestDecompressBytesToWriterRejectsOversizedOutput(t *testing.T) {
	input := bytes.Repeat([]byte("a"), 1024*1024)
	compressed, err := CompressBytes(input)
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	err = DecompressBytesToWriter(compressed, &output, 1024)
	if !errors.Is(err, ErrDecompressedDataTooLarge) {
		t.Fatalf("error = %v, want %v", err, ErrDecompressedDataTooLarge)
	}
	if output.Len() != 1025 {
		t.Fatalf("written bytes = %d, want 1025-byte bounded probe", output.Len())
	}
}
