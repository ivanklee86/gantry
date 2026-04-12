package build_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRootCommand is pulled from the main package via a test helper in the root.
// We re-use the exported constructor via an import alias so this package stays self-contained.
// The test binary is built with all packages, so we can call into main directly.

// runBuild executes `gantry build <args>` using the real root command and
// returns stdout, stderr, and any error.
func runBuild(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	// Import the real root command from the main package via the test binary.
	// We do this by invoking the real NewRootCommand from main_test helpers.
	// Since we cannot import `main` directly, we use os/exec to call the compiled binary.
	// For unit-style e2e tests we instead compile and run inline using cobra directly.
	//
	// This test package calls NewRootCommand from the module root via the _test build.
	// We register the build subcommand inside NewRootCommand, so these tests exercise
	// the full command tree without needing a separate binary.
	cmd := newTestRootCommand()
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(append([]string{"build"}, args...))
	runErr := cmd.Execute()
	return outBuf.String(), errBuf.String(), runErr
}

// --- Flag validation ---

func TestE2E_Build_MissingRepo_Error(t *testing.T) {
	_, _, err := runBuild(t, "--files", "a.json")
	assert.Error(t, err)
}

func TestE2E_Build_BothRefAndCommit_Error(t *testing.T) {
	_, _, err := runBuild(t,
		"--repo", "https://github.com/org/repo",
		"--ref", "refs/heads/main",
		"--commit", "abc1234",
		"--files", "a.json",
	)
	assert.Error(t, err)
}

func TestE2E_Build_NoFiles_Error(t *testing.T) {
	_, _, err := runBuild(t, "--repo", "https://github.com/org/repo")
	assert.Error(t, err)
}

// --- Config file ---

func TestE2E_Build_Config_YAML_LocalRepo(t *testing.T) {
	// renovate.json is a real JSON file at the repo root; valid input for the merger.
	yamlContent := `
version: 1
overlays:
  - repo: "/workspaces/gantry"
    files:
      - "renovate.json"
`
	configPath := writeTempConfig(t, yamlContent)
	stdout, _, err := runBuild(t, "--config", configPath)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &m))
	assert.NotEmpty(t, m)
}

func TestE2E_Build_Config_YAML_FileNotFound(t *testing.T) {
	_, _, err := runBuild(t, "--config", "/nonexistent/config.yaml")
	assert.Error(t, err)
}

// --- Write flag ---

func TestE2E_Build_Write_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "devcontainer.json")

	yamlContent := `
version: 1
output_path: "` + outPath + `"
overlays:
  - repo: "/workspaces/gantry"
    files:
      - "renovate.json"
`
	configPath := writeTempConfig(t, yamlContent)
	_, _, err := runBuild(t, "--config", configPath, "--write")
	require.NoError(t, err)
	assert.FileExists(t, outPath)
}

func TestE2E_Build_OutputPath_Override(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.json")

	yamlContent := `
version: 1
output_path: "/this/should/be/overridden.json"
overlays:
  - repo: "/workspaces/gantry"
    files:
      - "renovate.json"
`
	configPath := writeTempConfig(t, yamlContent)
	_, _, err := runBuild(t, "--config", configPath, "--write", "--output-path", outPath)
	require.NoError(t, err)
	assert.FileExists(t, outPath)
	assert.NoFileExists(t, "/this/should/be/overridden.json")
}

// --- Integration tests (requires network) ---

func TestIntegration_E2E_Build_RemoteRepo(t *testing.T) {
	if os.Getenv("GANTRY_INTEGRATION_TESTS") == "" {
		t.Skip("set GANTRY_INTEGRATION_TESTS=1 to run integration tests")
	}

	stdout, _, err := runBuild(t,
		"--repo", "https://github.com/ivanklee86/devcontainers//devcontainer_configs/bases/python",
		"--files", "devcontainer.json",
	)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &m))
	assert.NotEmpty(t, m)
}

func TestIntegration_E2E_Build_Config_MultiOverlay(t *testing.T) {
	if os.Getenv("GANTRY_INTEGRATION_TESTS") == "" {
		t.Skip("set GANTRY_INTEGRATION_TESTS=1 to run integration tests")
	}

	yamlContent := `
version: 1
overlays:
  - repo: "https://github.com/ivanklee86/devcontainers"
    subdirectory: "devcontainer_configs/bases/python"
    files:
      - "devcontainer.json"
  - repo: "https://github.com/ivanklee86/devcontainers"
    subdirectory: "devcontainer_configs/bases/python"
    files:
      - "devcontainer.json"
`
	configPath := writeTempConfig(t, yamlContent)
	stdout, _, err := runBuild(t, "--config", configPath)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &m))
	assert.NotEmpty(t, m)
}

// writeTempConfig writes content to a temp YAML file and returns the path.
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}
