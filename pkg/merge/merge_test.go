package merge_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ivanklee86/gantry/pkg/git"
	"github.com/ivanklee86/gantry/pkg/merge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerge_TwoJSONFiles(t *testing.T) {
	files := []git.FileContent{
		{Path: "a.json", Content: []byte(`{"x": 1, "y": "hello"}`)},
		{Path: "b.json", Content: []byte(`{"y": "world", "z": 2}`)},
	}
	result, err := merge.Merge(files)
	require.NoError(t, err)

	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &out))
	assert.Equal(t, float64(1), out["x"])
	assert.Equal(t, "world", out["y"]) // rightmost wins
	assert.Equal(t, float64(2), out["z"])
}

func TestMerge_JSONandJsonnet(t *testing.T) {
	files := []git.FileContent{
		{Path: "base.json", Content: []byte(`{"name": "myapp", "version": 1}`)},
		{Path: "overlay.jsonnet", Content: []byte(`{ version: std.length("hello") }`)},
	}
	result, err := merge.Merge(files)
	require.NoError(t, err)

	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &out))
	assert.Equal(t, "myapp", out["name"])
	assert.Equal(t, float64(5), out["version"]) // std.length("hello") == 5, not a string
}

func TestMerge_ThreeFiles(t *testing.T) {
	files := []git.FileContent{
		{Path: "a.json", Content: []byte(`{"x": 1, "y": 1, "z": 1}`)},
		{Path: "b.json", Content: []byte(`{"y": 2, "w": 2}`)},
		{Path: "c.json", Content: []byte(`{"z": 3}`)},
	}
	result, err := merge.Merge(files)
	require.NoError(t, err)

	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &out))
	assert.Equal(t, float64(1), out["x"])
	assert.Equal(t, float64(2), out["y"]) // b wins over a
	assert.Equal(t, float64(3), out["z"]) // c wins over a
	assert.Equal(t, float64(2), out["w"])
}

func TestMerge_DeepMerge(t *testing.T) {
	files := []git.FileContent{
		{Path: "base.json", Content: []byte(`{"settings": {"a": 1, "b": 2}}`)},
		{Path: "overlay.jsonnet", Content: []byte(`{ settings+: { b: 99, c: 3 } }`)},
	}
	result, err := merge.Merge(files)
	require.NoError(t, err)

	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &out))
	settings := out["settings"].(map[string]interface{})
	assert.Equal(t, float64(1), settings["a"])  // base key survived
	assert.Equal(t, float64(99), settings["b"]) // overlay wins
	assert.Equal(t, float64(3), settings["c"])  // overlay added
}

func TestMerge_InvalidJsonnet(t *testing.T) {
	files := []git.FileContent{
		{Path: "a.json", Content: []byte(`{"x": 1}`)},
		{Path: "b.jsonnet", Content: []byte(`{ broken syntax !!!`)},
	}
	_, err := merge.Merge(files)
	assert.Error(t, err)
}

func TestMerge_TooFewFiles(t *testing.T) {
	_, err := merge.Merge([]git.FileContent{})
	assert.Error(t, err)

	_, err = merge.Merge([]git.FileContent{
		{Path: "a.json", Content: []byte(`{"x": 1}`)},
	})
	assert.Error(t, err)
}

func TestMerge_EmptyPath(t *testing.T) {
	files := []git.FileContent{
		{Path: "a.json", Content: []byte(`{"x": 1}`)},
		{Path: "", Content: []byte(`{"y": 2}`)},
	}
	_, err := merge.Merge(files)
	assert.Error(t, err)
}

func TestMerge_DuplicatePaths(t *testing.T) {
	files := []git.FileContent{
		{Path: "a.json", Content: []byte(`{"x": 1}`)},
		{Path: "a.json", Content: []byte(`{"y": 2}`)},
	}
	_, err := merge.Merge(files)
	assert.Error(t, err)
}

func TestMerge_OutputIsValidJSON(t *testing.T) {
	files := []git.FileContent{
		{Path: "a.json", Content: []byte(`{"x": 1}`)},
		{Path: "b.json", Content: []byte(`{"y": 2}`)},
	}
	result, err := merge.Merge(files)
	require.NoError(t, err)

	var out interface{}
	assert.NoError(t, json.Unmarshal(result, &out))
}

func TestIntegration_Merge_DevcontainerWithJsonnetOverlay(t *testing.T) {
	if os.Getenv("GANTRY_INTEGRATION_TESTS") == "" {
		t.Skip("set GANTRY_INTEGRATION_TESTS=1 to run integration tests")
	}

	base, err := os.ReadFile("testdata/base.json")
	require.NoError(t, err)

	overlay, err := os.ReadFile("testdata/overlay.jsonnet")
	require.NoError(t, err)

	files := []git.FileContent{
		{Path: "base.json", Content: base},
		{Path: "overlay.jsonnet", Content: overlay},
	}

	result, err := merge.Merge(files)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &out))

	// Base fields survived.
	assert.Equal(t, "python-base", out["name"])
	assert.Equal(t, "vscode", out["remoteUser"])

	// Overlay added new extensions while base extensions survived (deep merge).
	vscode := out["customizations"].(map[string]interface{})["vscode"].(map[string]interface{})
	exts := vscode["extensions"].([]interface{})
	extStrings := make([]string, len(exts))
	for i, e := range exts {
		extStrings[i] = e.(string)
	}
	assert.Contains(t, extStrings, "charliermarsh.ruff")
	assert.Contains(t, extStrings, "astral-sh.ty")
	assert.Contains(t, extStrings, "ms-python.python") // from base, survived

	// Overlay added new postCreateCommand while base command survived.
	cmds := out["postCreateCommand"].(map[string]interface{})
	assert.Contains(t, cmds, "install-deps") // from base
	assert.Contains(t, cmds, "install-prek") // from overlay
}
