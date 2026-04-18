---
name: Project conventions and architecture
description: Test tiers, task runner, key packages, and architectural patterns in gantry
type: project
---

Three mandatory test tiers per CLAUDE.md:
- Unit tests: use mocked interfaces, in-package or `_test` package
- Integration tests: guarded by `GANTRY_INTEGRATION_TESTS=1` env var, naming convention `TestIntegration_*`, run via `task integration-test`
- e2e tests: call CLI via cobra command tree (not binary exec); implemented in cmd/build/build_test.go as of 2026-04-13

**Why:** Spec-mandated test structure; integration tests hit real network/files, so they must be gated.

**How to apply:** Any new package should have all three tiers. Integration tests use `t.Skip` when env var is absent.

Task runner is `Taskfile.yaml`. Test report is `tests.xml` (JUnit). `go-jsonnet v0.22.0` is a direct dependency via `pkg/merge`. Reference repo for devcontainer fixtures: `https://github.com/ivanklee86/devcontainers`.

Key architectural decisions:
- Package-level var `openRepoFunc` in pkg/gantry/build.go is the intentional injection point for mocking git in unit tests.
- build.go uses stripansi.Strip on user-supplied strings before log messages — defense against ANSI injection.
- exampleConfig() helper in helpers_test.go rewrites `repo: "./"` → absolute repoRoot for test runs.
- writeTempConfig() is intentionally duplicated across pkg/config/config_test.go (as writeTemp) and cmd/build/build_test.go — different packages, slight name difference.
