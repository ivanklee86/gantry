package gantry

import (
	"bytes"
	"testing"
)

func TestGantryHappyPath(t *testing.T) {
	b := bytes.NewBufferString("")

	gantry := New()
	if gantry == nil {
		t.Fatalf("New() returned nil gantry")
	}

	gantry.Out = b
	gantry.Err = b

	if gantry.Out != b {
		t.Fatalf("expected gantry.Out to be %T, got %T", b, gantry.Out)
	}
	if gantry.Err != b {
		t.Fatalf("expected gantry.Err to be %T, got %T", b, gantry.Err)
	}

	if gantry.Out != b {
		t.Errorf("expected gantry.Out to be set to buffer")
	}
	if gantry.Err != b {
		t.Errorf("expected gantry.Err to be set to buffer")
	}
}
