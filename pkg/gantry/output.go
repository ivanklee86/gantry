package gantry

import (
	"fmt"
	"io"
	"os"

	"github.com/jedib0t/go-pretty/v6/text"
)

const (
	headerPrefix = "gantry"
)

// printToStream prints a generic message to the provided stream (for example, stdout or stderr).
func printToStream(stream io.Writer, msg interface{}) {
	_, err := fmt.Fprintf(stream, "%v\n", msg)
	if err != nil {
		panic(err)
	}
}

// printToStreamWithColor prints a message after wrapping it in ANSI color codes.
func printToStreamWithColor(stream io.Writer, color text.Color, msg interface{}) {
	_, err := fmt.Fprint(stream, color.Sprintf("%v\n", msg))
	if err != nil {
		panic(err)
	}
}

// OutputHeading prints a header to stdout.
func (gantry Gantry) OutputHeading(msg interface{}) {
	printToStreamWithColor(gantry.Out, text.FgHiCyan, fmt.Sprintf("%v: %v", headerPrefix, msg))
}

// Output prints a normal message to stdout.
func (gantry Gantry) Output(msg interface{}) {
	printToStream(gantry.Out, msg)
}

// Error prints an error to stderr and exits with error code 1.
func (gantry Gantry) Error(msg interface{}) {
	printToStreamWithColor(gantry.Err, text.FgHiRed, fmt.Sprintf("Error: %v", msg))
	if !gantry.NoExitCode {
		os.Exit(1)
	}
}
