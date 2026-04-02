package gantry

import (
	"io"
	"os"
)

// Gantry is the logic/orchestrator.
type Gantry struct {
	// Allow swapping out stdout/stderr for testing.
	Out io.Writer
	Err io.Writer
}

// New returns a new instance of Gantry.
func New() *Gantry {
	return &Gantry{
		Out: os.Stdout,
		Err: os.Stderr,
	}
}
