package cli

import (
	"errors"
	"testing"
)

func TestPasswordResultRejectsEmptyPassword(t *testing.T) {
	_, err := passwordResult(passwordModel{done: true, value: ""})
	if !errors.Is(err, ErrEmptyPassword) {
		t.Fatalf("error = %v, want %v", err, ErrEmptyPassword)
	}
}

func TestPasswordResultReturnsEnteredPassword(t *testing.T) {
	got, err := passwordResult(passwordModel{done: true, value: "password"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "password" {
		t.Fatalf("password = %q, want %q", got, "password")
	}
}
