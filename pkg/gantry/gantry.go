package gantry

import (
	"io"
	"os"
)

type Config struct {
	NoExitCode bool
}

// Gantry is the logic/orchestrator.
type Gantry struct {
	*Config

	// Allow swapping out stdout/stderr for testing.
	Out io.Writer
	Err io.Writer
}

// New returns a new instance of Gantry.
func New() *Gantry {
	config := Config{}

	return &Gantry{
		Config: &config,
		Out:    os.Stdout,
		Err:    os.Stderr,
	}
}
