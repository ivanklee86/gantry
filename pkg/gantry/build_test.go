package gantry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ivanklee86/gantry/pkg/config"
	"github.com/ivanklee86/gantry/pkg/git"
)

// --- mockRepository ---

// mockRepository is a test double for git.Repository.
type mockRepository struct {
	files map[string][]byte
	err   error
}

func (m *mockRepository) GetFiles(paths []string) ([]git.FileContent, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := make([]git.FileContent, 0, len(paths))
	for _, p := range paths {
		content, ok := m.files[p]
		if !ok {
			return nil, errors.New("file not found: " + p)
		}
		out = append(out, git.FileContent{Path: p, Content: content})
	}
	return out, nil
}

// testGantry returns a Gantry with stdout/stderr wired to the provided buffers.
func testGantry(out, errBuf *bytes.Buffer) *Gantry {
	return &Gantry{Out: out, Err: errBuf}
}

// --- Build unit tests (using a patched openRepo) ---

// buildWithMocks calls Build but substitutes the provided mock repositories
// instead of opening real git repos.
func buildWithMocks(t *testing.T, cfg config.BuildConfig, repos []*mockRepository) (string, error) {
	t.Helper()

	var out bytes.Buffer
	var errBuf bytes.Buffer
	g := testGantry(&out, &errBuf)

	// Patch openRepo to return mocks in order.
	idx := 0
	origOpenRepo := openRepoFunc
	openRepoFunc = func(_ *Gantry, _ config.Overlay) (git.Repository, error) {
		if idx >= len(repos) {
			return nil, errors.New("no mock for overlay")
		}
		r := repos[idx]
		idx++
		return r, nil
	}
	t.Cleanup(func() { openRepoFunc = origOpenRepo })

	err := g.Build(context.Background(), cfg)
	return out.String(), err
}

func TestBuild_SingleOverlay_StdoutOutput(t *testing.T) {
	cfg := config.BuildConfig{
		Version: 1,
		Overlays: []config.Overlay{
			{Repo: "fake", Files: []string{"base.json"}},
		},
	}
	repos := []*mockRepository{
		{files: map[string][]byte{"base.json": []byte(`{"name":"python"}`)}},
	}

	out, err := buildWithMocks(t, cfg, repos)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &m))
	assert.Equal(t, "python", m["name"])
}

func TestBuild_MultipleOverlays_MergedOutput(t *testing.T) {
	cfg := config.BuildConfig{
		Version: 1,
		Overlays: []config.Overlay{
			{Repo: "fake1", Files: []string{"base.json"}},
			{Repo: "fake2", Files: []string{"override.json"}},
		},
	}
	repos := []*mockRepository{
		{files: map[string][]byte{"base.json": []byte(`{"name":"base","port":8080}`)}},
		{files: map[string][]byte{"override.json": []byte(`{"name":"override"}`)}},
	}

	out, err := buildWithMocks(t, cfg, repos)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &m))
	assert.Equal(t, "override", m["name"])
	assert.Equal(t, float64(8080), m["port"])
}

func TestBuild_WriteToFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "devcontainer.json")

	cfg := config.BuildConfig{
		Version:    1,
		Write:      true,
		OutputPath: outPath,
		Overlays: []config.Overlay{
			{Repo: "fake", Files: []string{"base.json"}},
		},
	}
	repos := []*mockRepository{
		{files: map[string][]byte{"base.json": []byte(`{"name":"test"}`)}},
	}

	_, err := buildWithMocks(t, cfg, repos)
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Equal(t, "test", m["name"])
}

func TestBuild_MkdirAll_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "nested", "deep", "devcontainer.json")

	cfg := config.BuildConfig{
		Version:    1,
		Write:      true,
		OutputPath: outPath,
		Overlays: []config.Overlay{
			{Repo: "fake", Files: []string{"base.json"}},
		},
	}
	repos := []*mockRepository{
		{files: map[string][]byte{"base.json": []byte(`{"key":"val"}`)}},
	}

	_, err := buildWithMocks(t, cfg, repos)
	require.NoError(t, err)
	assert.FileExists(t, outPath)
}

func TestBuild_RepoError_PropagatesError(t *testing.T) {
	cfg := config.BuildConfig{
		Version: 1,
		Overlays: []config.Overlay{
			{Repo: "fake", Files: []string{"missing.json"}},
		},
	}
	repos := []*mockRepository{
		{err: errors.New("clone failed")},
	}

	_, err := buildWithMocks(t, cfg, repos)
	assert.Error(t, err)
}

func TestBuild_GetFilesError_PropagatesError(t *testing.T) {
	cfg := config.BuildConfig{
		Version: 1,
		Overlays: []config.Overlay{
			{Repo: "fake", Files: []string{"missing.json"}},
		},
	}
	repos := []*mockRepository{
		{files: map[string][]byte{}}, // file not present
	}

	_, err := buildWithMocks(t, cfg, repos)
	assert.Error(t, err)
}

func TestBuild_MergeError_ReportsOverlayIndex(t *testing.T) {
	cfg := config.BuildConfig{
		Version: 1,
		Overlays: []config.Overlay{
			{Repo: "fake1", Files: []string{"base.json"}},
			{Repo: "fake2", Files: []string{"bad.jsonnet"}},
			{Repo: "fake3", Files: []string{"extra.json"}},
		},
	}
	repos := []*mockRepository{
		{files: map[string][]byte{"base.json": []byte(`{"name":"base"}`)}},
		{files: map[string][]byte{"bad.jsonnet": []byte(`{ broken syntax !!!`)}},
		{files: map[string][]byte{"extra.json": []byte(`{"extra":true}`)}},
	}

	_, err := buildWithMocks(t, cfg, repos)
	require.Error(t, err)
	assert.ErrorContains(t, err, "overlay 2")
}

// --- Integration tests ---

func TestIntegration_Build_LocalRepo_Stdout(t *testing.T) {
	if os.Getenv("GANTRY_INTEGRATION_TESTS") == "" {
		t.Skip("set GANTRY_INTEGRATION_TESTS=1 to run integration tests")
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	g := testGantry(&out, &errBuf)

	repoRoot, err := filepath.Abs("../../")
	require.NoError(t, err)

	cfg := config.BuildConfig{
		Version: 1,
		Overlays: []config.Overlay{
			{
				Repo:  repoRoot,
				Files: []string{"go.mod"},
			},
		},
	}

	err = g.Build(context.Background(), cfg)
	require.NoError(t, err)
	// go.mod is not valid JSON, so the merger will error — skip JSON check.
	// Just verify no panic and the file content is in the output.
	assert.Contains(t, out.String(), "ivanklee86/gantry")
}

func TestIntegration_Build_RemoteRepo_HTTPS(t *testing.T) {
	if os.Getenv("GANTRY_INTEGRATION_TESTS") == "" {
		t.Skip("set GANTRY_INTEGRATION_TESTS=1 to run integration tests")
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	g := testGantry(&out, &errBuf)

	cfg := config.BuildConfig{
		Version: 1,
		Overlays: []config.Overlay{
			{
				Repo:         "https://github.com/ivanklee86/devcontainers",
				Subdirectory: "devcontainer_configs/bases/python",
				Files:        []string{"devcontainer.json"},
			},
		},
	}

	err := g.Build(context.Background(), cfg)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(out.Bytes(), &m))
	assert.NotEmpty(t, m)
}

func TestIntegration_Build_MultipleOverlays_Remote(t *testing.T) {
	if os.Getenv("GANTRY_INTEGRATION_TESTS") == "" {
		t.Skip("set GANTRY_INTEGRATION_TESTS=1 to run integration tests")
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	g := testGantry(&out, &errBuf)

	cfg := config.BuildConfig{
		Version: 1,
		Overlays: []config.Overlay{
			{
				Repo:         "https://github.com/ivanklee86/devcontainers",
				Subdirectory: "devcontainer_configs/bases/python",
				Files:        []string{"devcontainer.json"},
			},
			{
				Repo:         "https://github.com/ivanklee86/devcontainers",
				Subdirectory: "devcontainer_configs/bases/python",
				Files:        []string{"devcontainer.json"},
			},
		},
	}

	err := g.Build(context.Background(), cfg)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(out.Bytes(), &m))
	assert.NotEmpty(t, m)
}
