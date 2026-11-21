package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoot(t *testing.T) {
	b := bytes.NewBufferString("")

	command := NewRootCommand()
	command.SetOut(b)
	command.SetArgs([]string{})
	err := command.Execute()
	if err != nil {
		t.Fatal(err)
	}

	out, err := io.ReadAll(b)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Print(string(out))

	assert.Contains(t, string(out), "gantry")
	assert.Contains(t, string(out), "build", "root usage lists the build subcommand")
}

func TestRoot_HelpDescribesGantry(t *testing.T) {
	b := bytes.NewBufferString("")

	command := NewRootCommand()
	command.SetOut(b)
	command.SetArgs([]string{"--help"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, b.String(), "devcontainer")
	assert.NotContains(t, b.String(), "minimum versions", "help text must describe gantry, not another CLI")
}
