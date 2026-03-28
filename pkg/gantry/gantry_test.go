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
		t.Errorf("expected gantry.Out to be set to buffer")
	}
	if gantry.Err != b {
		t.Errorf("expected gantry.Err to be set to buffer")
	}
}
