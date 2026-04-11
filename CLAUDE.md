# Development

- Commands for common development tasks (testing, linting, building) can be found in `Taskfile.yaml`
- The CLI should have three tiers of tests:
  - Unit tests (test code with mocked interfaces)
  - Integration tests (use local files, pull public repositories, etc to test isolated functions)
  - e2e tests (call CLI)
- You can use `https://github.com/ivanklee86/devcontainers` for a reference repository.

# Specifications

- Project specifications can be found in `docs/specs`.
