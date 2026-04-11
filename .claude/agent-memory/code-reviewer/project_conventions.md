---
name: Project conventions and architecture
description: Test tiers, task runner, key packages, and architectural patterns in gantry
type: project
---

Three mandatory test tiers per CLAUDE.md:
- Unit tests: use mocked interfaces, in-package or `_test` package
- Integration tests: guarded by `GANTRY_INTEGRATION_TESTS=1` env var, naming convention `TestIntegration_*`, run via `task integration-test`
- e2e tests: call the CLI binary directly (none exist yet as of 2026-04-11)

**Why:** Spec-mandated test structure; integration tests hit real network/files, so they must be gated.

**How to apply:** Any new package should have all three tiers. Absence of e2e tests is a known gap. Integration tests use `t.Skip` when env var is absent.

Task runner is `Taskfile.yaml`. Test report is `tests.xml` (JUnit). `go-jsonnet v0.22.0` is an *indirect* dependency — `pkg/merge` should promote it to a direct dependency in go.mod. Reference repo for devcontainer fixtures: `https://github.com/ivanklee86/devcontainers`.
