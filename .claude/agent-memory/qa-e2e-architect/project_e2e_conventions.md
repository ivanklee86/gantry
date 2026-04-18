---
name: Gantry e2e test conventions
description: File locations, helpers, patterns, and behavioral facts for Gantry e2e tests — essential for writing new tests correctly
type: project
---

## Key file locations

- e2e tests: `/workspaces/gantry/cmd/build/build_test.go`
- Test helpers: `/workspaces/gantry/cmd/build/helpers_test.go`
- Core build logic: `/workspaces/gantry/pkg/gantry/build.go`
- Config loading/validation: `/workspaces/gantry/pkg/config/config.go`
- Merge engine: `/workspaces/gantry/pkg/merge/merge.go`
- Git layer: `/workspaces/gantry/pkg/git/git.go`
- Example configs: `/workspaces/gantry/examples/local/`, `/workspaces/gantry/examples/errors/`

## Test helpers (from helpers_test.go)

- `runBuild(t, args...)` — invokes `gantry build <args>` via `newTestRootCommand()` in-process; returns `(stdout, stderr string, err error)`
- `exampleConfig(t, relPath)` — reads example file, replaces `repo: "./"` with absolute `repoRoot`, writes to temp file, returns path
- `writeTempConfig(t, content)` — writes inline YAML string to temp file, returns path
- `repoRoot` — init-time variable; absolute path to the repo (found by walking up to go.mod)
- All tests in package `build_test`

## CLI invocation pattern

Tests call `runBuild(t, "--flag", "value", ...)` — no binary compilation needed; uses Cobra root command in-process.
stdout/stderr are separate buffers; error returned when `cmd.Execute()` returns non-nil.

## stdout output behavior

`g.Output(string(merged))` calls `printToStream` which does `fmt.Fprintf(stream, "%v\n", msg)`.
The merged bytes from `merge.Merge()` have trailing newlines stripped (`strings.TrimRight(result, "\n")`).
So stdout will be: `<json>\n` (one newline added by Output). Valid JSON parsing with `json.Unmarshal([]byte(stdout), &m)` works fine.

## stderr content

All progress messages go to stderr (cyan). Error messages go to stderr (red, prefixed "Error: ").
On merge failure with a prior good result: stderr gets a yellow "Last known good result (after overlay N):" block followed by the JSON, then the red error.

## Validation rules (config.Validate)

- `version` must equal 1; any other value → "unsupported config version N: only version 1 is supported"
- `Write == true && OutputPath == ""` → "--output-path is required when --write is set"
- `len(Overlays) == 0` → "at least one overlay is required"
- Per overlay: repo required, ref+commit mutually exclusive, at least one file required
- Overlay index in errors is 0-based in Validate but `build.go` error messages are 1-based ("overlay 1", "overlay 2")
- `Write` field has `yaml:"-"` — cannot be set via YAML, only via `--write` CLI flag

## Merge engine behavior (pkg/merge/merge.go)

- `safePath` regex: `^[a-zA-Z0-9_\-./]+$` — rejects paths with spaces, `..` components, special chars
- Path check is on the FileContent.Path field, NOT the full filesystem path
- `..` in a file path passed via `--files` or config `files:` will hit safePath rejection in merge (but securejoin in git layer may catch it first)
- maxOutputBytes: 10 MB
- MaxStack: 100
- Returns bytes with trailing newline stripped

## Local repo fixture

`/workspaces/gantry` is a valid git repo usable as `--repo`. Available files:
- `renovate.json` — valid JSON
- `go.mod` — valid Jsonnet (single-file)
- `examples/local/base.json`, `examples/local/overlay.jsonnet`
- `examples/errors/bad.jsonnet`, `examples/errors/valid-base.json`
- `pkg/merge/testdata/base.json`, `pkg/merge/testdata/overlay.jsonnet`

## Integration test gate

Network-requiring tests use `if os.Getenv("GANTRY_INTEGRATION_TESTS") == "" { t.Skip(...) }`.
Run via `task integration-test` (sets `GANTRY_INTEGRATION_TESTS=1`).

## writeOutput behavior

`writeOutput` calls `os.MkdirAll(filepath.Dir(path), 0o755)` before writing — parent dirs are created automatically.

## ENV var interpolation

`expandConfig` is called in `LoadYAMLConfig` after unmarshal. Uses `os.Expand`. Applies to: `output_path`, `repo`, `ref`, `commit`, `subdirectory`, and all auth fields. Does NOT apply to `version`, `files` list items, or YAML keys.

**Why:** Knowing exactly which fields expand prevents writing tests that assert expansion where it doesn't happen.
**How to apply:** ENV var tests should set the var, put `${VAR}` in an interpolated field, and assert the expanded value is used.
