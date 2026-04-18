package build_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
  - repo: "` + repoRoot + `"
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
  - repo: "` + repoRoot + `"
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
  - repo: "` + repoRoot + `"
    files:
      - "renovate.json"
`
	configPath := writeTempConfig(t, yamlContent)
	_, _, err := runBuild(t, "--config", configPath, "--write", "--output-path", outPath)
	require.NoError(t, err)
	assert.FileExists(t, outPath)
	assert.NoFileExists(t, "/this/should/be/overridden.json")
}

// --- Example config files ---

func TestE2E_Build_ExampleLocalConfig_HappyPath(t *testing.T) {
	stdout, _, err := runBuild(t, "--config", exampleConfig(t, "examples/local/config.yaml"))
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &m))
	assert.Equal(t, "my-project-team", m["name"]) // overlay wins
	assert.Equal(t, "vscode", m["remoteUser"])    // base value inherited

	customizations, ok := m["customizations"].(map[string]interface{})
	require.True(t, ok)
	vscode, ok := customizations["vscode"].(map[string]interface{})
	require.True(t, ok)
	exts, ok := vscode["extensions"].([]interface{})
	require.True(t, ok)
	var extStrings []string
	for _, e := range exts {
		extStrings = append(extStrings, e.(string))
	}
	assert.Contains(t, extStrings, "ms-vscode.vscode-json") // from base
	assert.Contains(t, extStrings, "eamodio.gitlens")       // from overlay
}

func TestE2E_Build_ExampleBadJsonnet_ReportsOverlay(t *testing.T) {
	_, _, err := runBuild(t, "--config", exampleConfig(t, "examples/errors/bad-jsonnet.yaml"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "overlay 2")
}

func TestE2E_Build_ExampleMissingFile_ReportsError(t *testing.T) {
	_, _, err := runBuild(t, "--config", exampleConfig(t, "examples/errors/missing-file.yaml"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "overlay 1")
}

// --- Happy path ---

func TestE2E_Build_CLIFlags_SingleFile(t *testing.T) {
	stdout, _, err := runBuild(t, "--repo", repoRoot, "--files", "renovate.json")
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &m))
	assert.NotEmpty(t, m)
}

func TestE2E_Build_CLIFlags_MultiFile_SingleOverlay(t *testing.T) {
	stdout, _, err := runBuild(t,
		"--repo", repoRoot,
		"--files", "examples/local/base.json",
		"--files", "examples/local/overlay.jsonnet",
	)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &m))
	assert.Equal(t, "my-project-team", m["name"]) // overlay wins
	assert.Equal(t, "vscode", m["remoteUser"])    // base value inherited
}

func TestE2E_Build_Stdout_TrailingNewline(t *testing.T) {
	stdout, _, err := runBuild(t, "--repo", repoRoot, "--files", "renovate.json")
	require.NoError(t, err)

	assert.True(t, strings.HasSuffix(stdout, "\n"), "stdout should end with a newline")
	trimmed := strings.TrimRight(stdout, "\n")
	assert.NotContains(t, trimmed, "\n\n", "stdout should not have a blank line before the final newline")

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(stdout)), &m))
	assert.NotEmpty(t, m)
}

func TestE2E_Build_SingleFile_ValidJSON(t *testing.T) {
	stdout, _, err := runBuild(t, "--repo", repoRoot, "--files", "examples/errors/valid-base.json")
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &m))
	assert.Equal(t, "base", m["name"])
}

func TestE2E_Build_Write_CreatesNestedParentDirs(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "a", "b", "c", "devcontainer.json")
	configPath := writeTempConfig(t, `
version: 1
overlays:
  - repo: "`+repoRoot+`"
    files:
      - "renovate.json"
`)
	_, _, err := runBuild(t, "--config", configPath, "--write", "--output-path", outPath)
	require.NoError(t, err)
	assert.FileExists(t, outPath)
}

func TestE2E_Build_ThreeOverlay_AccumulationOrder(t *testing.T) {
	configPath := writeTempConfig(t, `
version: 1
overlays:
  - repo: "`+repoRoot+`"
    subdirectory: "examples/errors"
    files:
      - "valid-base.json"
  - repo: "`+repoRoot+`"
    subdirectory: "examples/local"
    files:
      - "base.json"
  - repo: "`+repoRoot+`"
    subdirectory: "examples/local"
    files:
      - "overlay.jsonnet"
`)
	stdout, _, err := runBuild(t, "--config", configPath)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &m))
	assert.Equal(t, "my-project-team", m["name"])                              // overlay3 wins
	assert.Equal(t, "vscode", m["remoteUser"])                                 // from overlay2
	assert.Equal(t, "mcr.microsoft.com/devcontainers/base:ubuntu", m["image"]) // from overlay2
}

func TestE2E_Build_EnvVar_RepoPath_Interpolation(t *testing.T) {
	t.Setenv("GANTRY_TEST_REPO_H7", repoRoot)
	configPath := writeTempConfig(t, `
version: 1
overlays:
  - repo: "${GANTRY_TEST_REPO_H7}"
    files:
      - "renovate.json"
`)
	stdout, _, err := runBuild(t, "--config", configPath)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &m))
	assert.NotEmpty(t, m)
}

func TestE2E_Build_EnvVar_OutputPath_Interpolation(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "out.json")
	t.Setenv("GANTRY_TEST_OUT_H8", outPath)
	configPath := writeTempConfig(t, `
version: 1
output_path: "${GANTRY_TEST_OUT_H8}"
overlays:
  - repo: "`+repoRoot+`"
    files:
      - "renovate.json"
`)
	_, _, err := runBuild(t, "--config", configPath, "--write")
	require.NoError(t, err)
	assert.FileExists(t, outPath)
}

func TestE2E_Build_Subdirectory_FieldInYAML(t *testing.T) {
	configPath := writeTempConfig(t, `
version: 1
overlays:
  - repo: "`+repoRoot+`"
    subdirectory: "examples/local"
    files:
      - "base.json"
`)
	stdout, _, err := runBuild(t, "--config", configPath)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &m))
	assert.Equal(t, "vscode", m["remoteUser"])
	assert.Equal(t, "my-project", m["name"])
}

// --- Rainy day ---

func TestE2E_Build_WriteFlagWithoutOutputPath_Error(t *testing.T) {
	_, _, err := runBuild(t, "--repo", repoRoot, "--files", "renovate.json", "--write")
	require.Error(t, err)
	assert.ErrorContains(t, err, "--output-path")
}

func TestE2E_Build_UnsupportedVersion_Error(t *testing.T) {
	configPath := writeTempConfig(t, `
version: 2
overlays:
  - repo: "`+repoRoot+`"
    files:
      - "renovate.json"
`)
	_, _, err := runBuild(t, "--config", configPath)
	require.Error(t, err)
	assert.ErrorContains(t, err, "unsupported config version 2")
}

func TestE2E_Build_EmptyOverlays_Error(t *testing.T) {
	configPath := writeTempConfig(t, `
version: 1
overlays: []
`)
	_, _, err := runBuild(t, "--config", configPath)
	require.Error(t, err)
	assert.ErrorContains(t, err, "at least one overlay is required")
}

func TestE2E_Build_OverlayMissingFiles_Error(t *testing.T) {
	configPath := writeTempConfig(t, `
version: 1
overlays:
  - repo: "`+repoRoot+`"
`)
	_, _, err := runBuild(t, "--config", configPath)
	require.Error(t, err)
	assert.ErrorContains(t, err, "overlay 1")
	assert.ErrorContains(t, err, "at least one file is required")
}

func TestE2E_Build_OverlayMissingRepo_Error(t *testing.T) {
	configPath := writeTempConfig(t, `
version: 1
overlays:
  - files:
      - "renovate.json"
`)
	_, _, err := runBuild(t, "--config", configPath)
	require.Error(t, err)
	assert.ErrorContains(t, err, "overlay 1")
	assert.ErrorContains(t, err, "repo is required")
}

func TestE2E_Build_OverlayBothRefAndCommit_Error(t *testing.T) {
	configPath := writeTempConfig(t, `
version: 1
overlays:
  - repo: "https://github.com/org/repo"
    ref: "refs/heads/main"
    commit: "abc1234567"
    files:
      - "README.md"
`)
	_, _, err := runBuild(t, "--config", configPath)
	require.Error(t, err)
	assert.ErrorContains(t, err, "overlay 1")
	assert.ErrorContains(t, err, "ref and commit are mutually exclusive")
}

func TestE2E_Build_MergeError_FirstOverlay_NoLastKnownGood(t *testing.T) {
	configPath := writeTempConfig(t, `
version: 1
overlays:
  - repo: "`+repoRoot+`"
    subdirectory: "examples/errors"
    files:
      - "bad.jsonnet"
`)
	_, stderr, err := runBuild(t, "--config", configPath)
	require.Error(t, err)
	assert.ErrorContains(t, err, "overlay 1")
	assert.NotContains(t, stderr, "Last known good result")
}

func TestE2E_Build_MergeError_ThirdOverlay_PrintsLastKnownGood(t *testing.T) {
	configPath := writeTempConfig(t, `
version: 1
overlays:
  - repo: "`+repoRoot+`"
    subdirectory: "examples/errors"
    files:
      - "valid-base.json"
  - repo: "`+repoRoot+`"
    subdirectory: "examples/local"
    files:
      - "base.json"
  - repo: "`+repoRoot+`"
    subdirectory: "examples/errors"
    files:
      - "bad.jsonnet"
`)
	_, stderr, err := runBuild(t, "--config", configPath)
	require.Error(t, err)
	assert.ErrorContains(t, err, "overlay 3")
	assert.Contains(t, stderr, "Last known good result (after overlay 2):")

	// The last-known-good JSON appears immediately after the "last known good" line in stderr.
	idx := strings.Index(stderr, "{")
	require.NotEqual(t, -1, idx, "expected JSON object in stderr after last-known-good message")
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stderr[idx:strings.LastIndex(stderr, "}")+1]), &m))
	assert.NotEmpty(t, m)
}

func TestE2E_Build_WriteFieldInYAML_IsIgnored(t *testing.T) {
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "should-not-exist.json")
	configPath := writeTempConfig(t, `
version: 1
write: true
output_path: "`+sentinel+`"
overlays:
  - repo: "`+repoRoot+`"
    files:
      - "renovate.json"
`)
	stdout, _, err := runBuild(t, "--config", configPath)
	require.NoError(t, err)          // write: true in YAML is ignored (yaml:"-")
	assert.NotEmpty(t, stdout)       // output went to stdout
	assert.NoFileExists(t, sentinel) // file was NOT written
}

func TestE2E_Build_MalformedYAML_Error(t *testing.T) {
	configPath := writeTempConfig(t, `version: [not: valid yaml`)
	_, _, err := runBuild(t, "--config", configPath)
	require.Error(t, err)
	assert.ErrorContains(t, err, "parse config")
}

// --- Edge cases ---

func TestE2E_Build_DotDotPath_Rejected(t *testing.T) {
	// ../etc/passwd passes the safePath regex but securejoin clamps it within
	// the worktree, so the file is simply not found.
	_, _, err := runBuild(t, "--repo", repoRoot, "--files", "../etc/passwd")
	assert.Error(t, err)
}

func TestE2E_Build_PathWithSpaces_Rejected(t *testing.T) {
	// GetFiles fails first with "no such file" because the file doesn't exist in the
	// worktree. If the file did exist, merge.Merge would then reject it via safePath
	// with "invalid path characters". Either way, the build must error.
	_, _, err := runBuild(t, "--repo", repoRoot, "--files", "a file with spaces.json")
	assert.Error(t, err)
}

func TestE2E_Build_UndefinedEnvVar_ExpandsToEmpty_Error(t *testing.T) {
	// GANTRY_UNDEFINED_XYZ_99 is never set; os.Expand returns "" for unset vars.
	configPath := writeTempConfig(t, `
version: 1
overlays:
  - repo: "${GANTRY_UNDEFINED_XYZ_99}"
    files:
      - "renovate.json"
`)
	_, _, err := runBuild(t, "--config", configPath)
	require.Error(t, err)
	assert.ErrorContains(t, err, "repo is required")
}

func TestE2E_Build_FileURILocalRepo(t *testing.T) {
	// Tests the file:// URI scheme path through IsLocalPath → OpenLocal.
	// The repo is always at /workspaces/gantry in the devcontainer.
	configPath := writeTempConfig(t, `
version: 1
overlays:
  - repo: "file:///workspaces/gantry"
    files:
      - "renovate.json"
`)
	stdout, _, err := runBuild(t, "--config", configPath)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &m))
	assert.NotEmpty(t, m)
}

func TestE2E_Build_CanonicalJSONOutput(t *testing.T) {
	stdout, _, err := runBuild(t, "--repo", repoRoot, "--files", "renovate.json")
	require.NoError(t, err)
	assert.Contains(t, stdout, `"$schema"`) // key from renovate.json survived round-trip
	assert.NotContains(t, stdout, "\t")     // Jsonnet uses spaces, not tabs
}

func TestE2E_Build_OutputPathInYAML_WithoutWrite_NoFile(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "should-not-exist.json")
	configPath := writeTempConfig(t, `
version: 1
output_path: "`+outPath+`"
overlays:
  - repo: "`+repoRoot+`"
    files:
      - "renovate.json"
`)
	stdout, _, err := runBuild(t, "--config", configPath)
	require.NoError(t, err)
	assert.NotEmpty(t, stdout)
	assert.NoFileExists(t, outPath)
}

func TestE2E_Build_MultiFilesOrder_LastWins(t *testing.T) {
	// valid-base.json has name="base"; examples/local/base.json has name="my-project".
	// Second file's top-level fields win.
	stdout, _, err := runBuild(t,
		"--repo", repoRoot,
		"--files", "examples/errors/valid-base.json",
		"--files", "examples/local/base.json",
	)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &m))
	assert.Equal(t, "my-project", m["name"])
}

func TestE2E_Build_ConfigTakesPrecedenceOverFlags(t *testing.T) {
	// When --config is supplied, --repo and --files flags are ignored.
	// The config uses valid-base.json (name="base"); renovate.json has no "name" key.
	configPath := writeTempConfig(t, `
version: 1
overlays:
  - repo: "`+repoRoot+`"
    subdirectory: "examples/errors"
    files:
      - "valid-base.json"
`)
	stdout, _, err := runBuild(t,
		"--config", configPath,
		"--repo", repoRoot, "--files", "renovate.json", // ignored
	)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &m))
	assert.Equal(t, "base", m["name"]) // came from config, not from renovate.json
}

func TestE2E_Build_EnvVar_Subdirectory_Interpolation(t *testing.T) {
	t.Setenv("GANTRY_TEST_SUBDIR_E9", "examples/local")
	configPath := writeTempConfig(t, `
version: 1
overlays:
  - repo: "`+repoRoot+`"
    subdirectory: "${GANTRY_TEST_SUBDIR_E9}"
    files:
      - "base.json"
`)
	stdout, _, err := runBuild(t, "--config", configPath)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &m))
	assert.Equal(t, "vscode", m["remoteUser"])
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
