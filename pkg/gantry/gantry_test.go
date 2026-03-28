package gantry

import (
	"bytes"
	"testing"
)

func TestGantryHappyPath(t *testing.T) {
	b := bytes.NewBufferString("")

	gantry := New()
	gantry.Out = b
	gantry.Err = b
}
