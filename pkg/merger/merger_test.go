package merger

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ivanklee86/gantry/pkg/git"
)

// helpers

func files(pairs ...string) []git.FileContent {
	if len(pairs)%2 != 0 {
		panic("files: pairs must be even (path, content, path, content, ...)")
	}
	out := make([]git.FileContent, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		out[i/2] = git.FileContent{Path: pairs[i], Content: []byte(pairs[i+1])}
	}
	return out
}

func mustUnmarshal(t *testing.T, s string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(s), &m))
	return m
}

// --- Merge ---

func TestMerge_NoFiles_Error(t *testing.T) {
	_, err := Merge(nil)
	assert.Error(t, err)

	_, err = Merge([]git.FileContent{})
	assert.Error(t, err)
}

func TestMerge_SingleFile_ReturnsFormatted(t *testing.T) {
	result, err := Merge(files("base.json", `{"name":"python","version":3}`))
	require.NoError(t, err)

	m := mustUnmarshal(t, result)
	assert.Equal(t, "python", m["name"])
	assert.Equal(t, float64(3), m["version"])
}

func TestMerge_TwoFiles_LaterOverridesEarlier(t *testing.T) {
	fs := files(
		"base.json", `{"name":"base","port":8080}`,
		"override.json", `{"name":"override","extra":true}`,
	)
	result, err := Merge(fs)
	require.NoError(t, err)

	m := mustUnmarshal(t, result)
	assert.Equal(t, "override", m["name"])    // overridden
	assert.Equal(t, float64(8080), m["port"]) // inherited from base
	assert.Equal(t, true, m["extra"])         // added by override
}

func TestMerge_ThreeFiles_OrderMatters(t *testing.T) {
	fs := files(
		"a.json", `{"key":"a"}`,
		"b.json", `{"key":"b"}`,
		"c.json", `{"key":"c"}`,
	)
	result, err := Merge(fs)
	require.NoError(t, err)

	m := mustUnmarshal(t, result)
	assert.Equal(t, "c", m["key"])
}

func TestMerge_JsonnetExpression(t *testing.T) {
	fs := files(
		"base.jsonnet",
		`local name = "devcontainer"; {"name": name, "version": 1}`,
	)
	result, err := Merge(fs)
	require.NoError(t, err)

	m := mustUnmarshal(t, result)
	assert.Equal(t, "devcontainer", m["name"])
	assert.Equal(t, float64(1), m["version"])
}

func TestMerge_InvalidJSON_Error(t *testing.T) {
	_, err := Merge(files("bad.json", `{not valid json`))
	assert.Error(t, err)
}

func TestMerge_ImportDisabled_Error(t *testing.T) {
	// A Jsonnet file that tries to import a path not in the provided files.
	fs := files("main.jsonnet", `import "external.json"`)
	_, err := Merge(fs)
	assert.Error(t, err)
}

func TestMerge_ImportFromPreloaded_Works(t *testing.T) {
	// Two files where the second imports the first.
	fs := files(
		"base.json", `{"name":"base"}`,
		"overlay.jsonnet", `(import "base.json") + {"extra": true}`,
	)
	result, err := Merge(fs)
	require.NoError(t, err)

	m := mustUnmarshal(t, result)
	assert.Equal(t, "base", m["name"])
	assert.Equal(t, true, m["extra"])
}
