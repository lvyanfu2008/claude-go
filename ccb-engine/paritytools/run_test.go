package paritytools

import (
	"context"
	"errors"
	"testing"
)

func TestRun_ErrNotHandled(t *testing.T) {
	t.Parallel()
	_, _, err := Run(context.Background(), "Read", []byte("{}"), Config{})
	if !errors.Is(err, ErrNotHandled) {
		t.Fatalf("expected ErrNotHandled, got %v", err)
	}
}

func TestIsNotHandled(t *testing.T) {
	t.Parallel()
	if !IsNotHandled(ErrNotHandled) {
		t.Fatal()
	}
	if IsNotHandled(errors.New("x")) {
		t.Fatal()
	}
}
