package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// OverlayAuth holds authentication credentials for a single overlay's repository.
// All string fields support ${ENV_VAR} interpolation.
type OverlayAuth struct {
	Token          string `yaml:"token"`
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`
	SSHKeyPath     string `yaml:"ssh_key_path"`
	SSHKeyPassword string `yaml:"ssh_key_password"`
}

// Overlay describes a single source repository and the ordered list of files
// to extract from it for merging.
type Overlay struct {
	// Repo is a remote URL or local path. Remote URLs support https://, http://,
	// ssh://, git://, and git@ (SCP-style). Local paths may be absolute,
	// relative (./  ../), or file:// URIs.
	// Append //subdir to set a subdirectory (e.g. https://github.com/org/repo//configs).
	Repo string `yaml:"repo"`

	// Ref is the branch or tag reference (e.g. refs/heads/main).
	// Mutually exclusive with Commit.
	Ref string `yaml:"ref"`

	// Commit is an exact commit SHA to check out (7–40 hex characters).
	// Mutually exclusive with Ref.
	Commit string `yaml:"commit"`

	// Subdirectory is prepended to all file paths when reading from the repo.
	// For remote URLs the //subdir URL syntax may be used instead.
	Subdirectory string `yaml:"subdirectory"`

	// Files is the ordered list of file paths to extract and merge.
	Files []string `yaml:"files"`

	// Auth holds optional credentials for cloning the repository.
	Auth OverlayAuth `yaml:"auth"`
}

// BuildConfig is the canonical input type for Gantry.Build.
// It is produced either from a YAML file (LoadYAMLConfig) or from CLI flags (CLIFlagsToConfig).
type BuildConfig struct {
	// Version must be 1.
	Version int `yaml:"version"`

	// OutputPath is the destination path for the merged output.
	OutputPath string `yaml:"output_path"`

	// Write controls whether output is written to OutputPath (true) or stdout (false).
	// This field is not present in the YAML schema; it is set by the --write CLI flag.
	Write bool `yaml:"-"`

	// Overlays is the ordered list of repositories and files to merge.
	Overlays []Overlay `yaml:"overlays"`
}

// CLIFlags holds all values parsed from the `gantry build` flags for single-repo use.
type CLIFlags struct {
	Repo           string
	Ref            string
	Commit         string
	Files          []string
	OutputPath     string
	Write          bool
	Token          string
	Username       string
	Password       string
	SSHKeyPath     string
	SSHKeyPassword string
}

// LoadYAMLConfig reads a YAML config file, expands ${ENV_VAR} references in all
// string fields, and validates the result.
func LoadYAMLConfig(path string) (BuildConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return BuildConfig{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg BuildConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return BuildConfig{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	expandConfig(&cfg)

	if err := Validate(cfg); err != nil {
		return BuildConfig{}, err
	}

	return cfg, nil
}

// CLIFlagsToConfig builds a single-overlay BuildConfig from parsed CLI flags.
func CLIFlagsToConfig(f CLIFlags) (BuildConfig, error) {
	if f.Repo == "" {
		return BuildConfig{}, fmt.Errorf("--repo is required")
	}
	if f.Ref != "" && f.Commit != "" {
		return BuildConfig{}, fmt.Errorf("--ref and --commit are mutually exclusive")
	}
	if len(f.Files) == 0 {
		return BuildConfig{}, fmt.Errorf("at least one --files entry is required")
	}

	cfg := BuildConfig{
		Version:    1,
		OutputPath: f.OutputPath,
		Write:      f.Write,
		Overlays: []Overlay{
			{
				Repo:   f.Repo,
				Ref:    f.Ref,
				Commit: f.Commit,
				Files:  f.Files,
				Auth: OverlayAuth{
					Token:          f.Token,
					Username:       f.Username,
					Password:       f.Password,
					SSHKeyPath:     f.SSHKeyPath,
					SSHKeyPassword: f.SSHKeyPassword,
				},
			},
		},
	}
	return cfg, nil
}

// Validate checks that a BuildConfig is semantically valid.
func Validate(cfg BuildConfig) error {
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported config version %d: only version 1 is supported", cfg.Version)
	}
	if len(cfg.Overlays) == 0 {
		return fmt.Errorf("at least one overlay is required")
	}
	for i, o := range cfg.Overlays {
		if o.Repo == "" {
			return fmt.Errorf("overlay %d: repo is required", i)
		}
		if o.Ref != "" && o.Commit != "" {
			return fmt.Errorf("overlay %d: ref and commit are mutually exclusive", i)
		}
		if len(o.Files) == 0 {
			return fmt.Errorf("overlay %d: at least one file is required", i)
		}
	}
	return nil
}

// expandConfig expands ${ENV_VAR} references in all string fields of cfg in-place.
func expandConfig(cfg *BuildConfig) {
	cfg.OutputPath = expandEnvVars(cfg.OutputPath)
	for i := range cfg.Overlays {
		o := &cfg.Overlays[i]
		o.Repo = expandEnvVars(o.Repo)
		o.Ref = expandEnvVars(o.Ref)
		o.Commit = expandEnvVars(o.Commit)
		o.Subdirectory = expandEnvVars(o.Subdirectory)
		o.Auth.Token = expandEnvVars(o.Auth.Token)
		o.Auth.Username = expandEnvVars(o.Auth.Username)
		o.Auth.Password = expandEnvVars(o.Auth.Password)
		o.Auth.SSHKeyPath = expandTilde(expandEnvVars(o.Auth.SSHKeyPath))
		o.Auth.SSHKeyPassword = expandEnvVars(o.Auth.SSHKeyPassword)
	}
}

// expandEnvVars expands ${VAR} and $VAR references using the current environment.
func expandEnvVars(s string) string {
	return os.Expand(s, os.Getenv)
}

// expandTilde replaces a leading ~ with the current user's home directory.
func expandTilde(s string) string {
	if !strings.HasPrefix(s, "~") {
		return s
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return s
	}
	return filepath.Join(home, s[1:])
}
