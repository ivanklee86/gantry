package gantry

import (
	"bytes"
	"io"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/assert"
)

func TestOutputs(t *testing.T) {
	b := bytes.NewBufferString("")

	gantry := New()
	gantry.Out = b
	gantry.Err = b

	testPhrase := "I'm a little hamster."

	t.Run("outputs string", func(t *testing.T) {
		gantry.Output(testPhrase)

		out, err := io.ReadAll(b)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, testPhrase+"\n", string(out))
	})

	t.Run("outputs header", func(t *testing.T) {
		gantry.OutputHeading(testPhrase)

		out, err := io.ReadAll(b)
		if err != nil {
			t.Fatal(err)
		}

		assert.Contains(t, stripansi.Strip(string(out)), testPhrase)
	})
}
