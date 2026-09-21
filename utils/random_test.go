package utils

import (
	"bytes"
	"errors"
	"testing"
)

type failingReader struct {
	err error
}

func (r failingReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestGenerateUniqueRandomNamePropagatesRandomSourceFailure(t *testing.T) {
	wantErr := errors.New("random source unavailable")
	_, err := generateUniqueRandomName(map[string]string{}, failingReader{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestGenerateUniqueRandomNameRetriesCollision(t *testing.T) {
	first := bytes.Repeat([]byte{0}, 16)
	second := bytes.Repeat([]byte{1}, 16)
	random := bytes.NewReader(append(first, second...))
	existing := map[string]string{"aaaaaaaaaaaaaaaa": "existing"}

	got, err := generateUniqueRandomName(existing, random)
	if err != nil {
		t.Fatal(err)
	}
	if got != "bbbbbbbbbbbbbbbb" {
		t.Fatalf("name = %q, want %q", got, "bbbbbbbbbbbbbbbb")
	}
}
