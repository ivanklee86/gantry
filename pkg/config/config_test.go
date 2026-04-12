package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Validate ---

func TestValidate_Valid(t *testing.T) {
	cfg := BuildConfig{
		Version:    1,
		OutputPath: "out.json",
		Overlays: []Overlay{
			{Repo: "https://github.com/org/repo", Files: []string{"a.json"}},
		},
	}
	assert.NoError(t, Validate(cfg))
}

func TestValidate_VersionMustBeOne(t *testing.T) {
	cfg := BuildConfig{
		Version: 2,
		Overlays: []Overlay{
			{Repo: "https://github.com/org/repo", Files: []string{"a.json"}},
		},
	}
	assert.Error(t, Validate(cfg))
}

func TestValidate_NoOverlays(t *testing.T) {
	cfg := BuildConfig{Version: 1}
	assert.Error(t, Validate(cfg))
}

func TestValidate_MissingRepo(t *testing.T) {
	cfg := BuildConfig{
		Version:  1,
		Overlays: []Overlay{{Files: []string{"a.json"}}},
	}
	assert.Error(t, Validate(cfg))
}

func TestValidate_MutualExclusion_RefAndCommit(t *testing.T) {
	cfg := BuildConfig{
		Version: 1,
		Overlays: []Overlay{
			{Repo: "https://github.com/org/repo", Ref: "refs/heads/main", Commit: "abc1234", Files: []string{"a.json"}},
		},
	}
	assert.Error(t, Validate(cfg))
}

func TestValidate_EmptyFiles(t *testing.T) {
	cfg := BuildConfig{
		Version:  1,
		Overlays: []Overlay{{Repo: "https://github.com/org/repo"}},
	}
	assert.Error(t, Validate(cfg))
}

// --- expandEnvVars ---

func TestExpandEnvVars_DollarBrace(t *testing.T) {
	t.Setenv("TEST_TOKEN", "mytoken")
	assert.Equal(t, "mytoken", expandEnvVars("${TEST_TOKEN}"))
}

func TestExpandEnvVars_Dollar(t *testing.T) {
	t.Setenv("TEST_TOKEN", "mytoken")
	assert.Equal(t, "mytoken", expandEnvVars("$TEST_TOKEN"))
}

func TestExpandEnvVars_NoVar(t *testing.T) {
	assert.Equal(t, "literal", expandEnvVars("literal"))
}

func TestExpandEnvVars_UnsetVar(t *testing.T) {
	require.NoError(t, os.Unsetenv("GANTRY_UNSET_VAR_XYZ"))
	assert.Equal(t, "", expandEnvVars("${GANTRY_UNSET_VAR_XYZ}"))
}

// --- expandTilde ---

func TestExpandTilde_Expands(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	result := expandTilde("~/.ssh/id_rsa")
	assert.Equal(t, filepath.Join(home, ".ssh/id_rsa"), result)
}

func TestExpandTilde_NoTilde(t *testing.T) {
	assert.Equal(t, "/absolute/path", expandTilde("/absolute/path"))
}

// --- expandConfig ---

func TestExpandConfig_ExpandsAllFields(t *testing.T) {
	t.Setenv("TEST_OUTPUT", "/tmp/out.json")
	t.Setenv("TEST_REPO", "https://github.com/org/repo")
	t.Setenv("TEST_TOKEN", "tok123")

	cfg := BuildConfig{
		Version:    1,
		OutputPath: "${TEST_OUTPUT}",
		Overlays: []Overlay{
			{
				Repo:  "${TEST_REPO}",
				Files: []string{"a.json"},
				Auth:  OverlayAuth{Token: "${TEST_TOKEN}"},
			},
		},
	}
	expandConfig(&cfg)

	assert.Equal(t, "/tmp/out.json", cfg.OutputPath)
	assert.Equal(t, "https://github.com/org/repo", cfg.Overlays[0].Repo)
	assert.Equal(t, "tok123", cfg.Overlays[0].Auth.Token)
}

// --- CLIFlagsToConfig ---

func TestCLIFlagsToConfig_Basic(t *testing.T) {
	f := CLIFlags{
		Repo:       "https://github.com/org/repo",
		Files:      []string{"base.json", "override.json"},
		OutputPath: "out.json",
		Write:      true,
		Token:      "tok",
	}
	cfg, err := CLIFlagsToConfig(f)
	require.NoError(t, err)
	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, "out.json", cfg.OutputPath)
	assert.True(t, cfg.Write)
	require.Len(t, cfg.Overlays, 1)
	assert.Equal(t, "https://github.com/org/repo", cfg.Overlays[0].Repo)
	assert.Equal(t, []string{"base.json", "override.json"}, cfg.Overlays[0].Files)
	assert.Equal(t, "tok", cfg.Overlays[0].Auth.Token)
}

func TestCLIFlagsToConfig_MissingRepo(t *testing.T) {
	_, err := CLIFlagsToConfig(CLIFlags{Files: []string{"a.json"}})
	assert.Error(t, err)
}

func TestCLIFlagsToConfig_BothRefAndCommit_Error(t *testing.T) {
	_, err := CLIFlagsToConfig(CLIFlags{
		Repo:   "https://github.com/org/repo",
		Ref:    "refs/heads/main",
		Commit: "abc1234",
		Files:  []string{"a.json"},
	})
	assert.Error(t, err)
}

func TestCLIFlagsToConfig_NoFiles_Error(t *testing.T) {
	_, err := CLIFlagsToConfig(CLIFlags{Repo: "https://github.com/org/repo"})
	assert.Error(t, err)
}

// --- LoadYAMLConfig ---

func TestLoadYAMLConfig_ValidFull(t *testing.T) {
	t.Setenv("GANTRY_TEST_TOKEN", "secret")

	yamlContent := `
version: 1
output_path: "/tmp/devcontainer.json"
overlays:
  - repo: "https://github.com/org/repo"
    ref: "refs/heads/main"
    subdirectory: "configs"
    files:
      - "base.json"
      - "override.json"
    auth:
      token: "${GANTRY_TEST_TOKEN}"
  - repo: "./local"
    files:
      - "local.json"
`
	path := writeTemp(t, "config.yaml", yamlContent)
	cfg, err := LoadYAMLConfig(path)
	require.NoError(t, err)

	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, "/tmp/devcontainer.json", cfg.OutputPath)
	require.Len(t, cfg.Overlays, 2)

	o0 := cfg.Overlays[0]
	assert.Equal(t, "https://github.com/org/repo", o0.Repo)
	assert.Equal(t, "refs/heads/main", o0.Ref)
	assert.Equal(t, "configs", o0.Subdirectory)
	assert.Equal(t, []string{"base.json", "override.json"}, o0.Files)
	assert.Equal(t, "secret", o0.Auth.Token)

	o1 := cfg.Overlays[1]
	assert.Equal(t, "./local", o1.Repo)
}

func TestLoadYAMLConfig_EnvVarExpansion(t *testing.T) {
	t.Setenv("GANTRY_TEST_USER", "testuser")
	t.Setenv("GANTRY_TEST_PASS", "testpass")

	yamlContent := `
version: 1
output_path: "out.json"
overlays:
  - repo: "https://github.com/org/repo"
    files:
      - "a.json"
    auth:
      username: "${GANTRY_TEST_USER}"
      password: "${GANTRY_TEST_PASS}"
`
	path := writeTemp(t, "config.yaml", yamlContent)
	cfg, err := LoadYAMLConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "testuser", cfg.Overlays[0].Auth.Username)
	assert.Equal(t, "testpass", cfg.Overlays[0].Auth.Password)
}

func TestLoadYAMLConfig_InvalidVersion(t *testing.T) {
	yamlContent := `
version: 99
output_path: "out.json"
overlays:
  - repo: "https://github.com/org/repo"
    files:
      - "a.json"
`
	path := writeTemp(t, "config.yaml", yamlContent)
	_, err := LoadYAMLConfig(path)
	assert.Error(t, err)
}

func TestLoadYAMLConfig_FileNotFound(t *testing.T) {
	_, err := LoadYAMLConfig("/nonexistent/path/config.yaml")
	assert.Error(t, err)
}

func TestLoadYAMLConfig_InvalidYAML(t *testing.T) {
	path := writeTemp(t, "config.yaml", ":::invalid yaml:::")
	_, err := LoadYAMLConfig(path)
	assert.Error(t, err)
}

// --- Integration tests ---

func TestIntegration_LoadYAMLConfig_FromFile(t *testing.T) {
	if os.Getenv("GANTRY_INTEGRATION_TESTS") == "" {
		t.Skip("set GANTRY_INTEGRATION_TESTS=1 to run integration tests")
	}

	yamlContent := `
version: 1
output_path: "/tmp/devcontainer.json"
overlays:
  - repo: "https://github.com/ivanklee86/devcontainers"
    ref: "refs/heads/main"
    subdirectory: "devcontainer_configs/bases/python"
    files:
      - "devcontainer.json"
`
	path := writeTemp(t, "integration_config.yaml", yamlContent)
	cfg, err := LoadYAMLConfig(path)
	require.NoError(t, err)
	assert.Equal(t, 1, cfg.Version)
	assert.Len(t, cfg.Overlays, 1)
}

// writeTemp creates a temporary file with the given name and content and returns its path.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}
