---
name: Git module security architecture
description: Security-relevant architecture decisions and vulnerability patterns found in the git module (pkg/git/git.go)
type: project
---

The git module clones remote repos into memory (go-git in-memory filesystem) and reads local repos via go-git's PlainOpen. Key security-relevant decisions:

- validateURL allowlist: https, http, ssh, git, git@ — file:// explicitly blocked in validateURL but NOT blocked in OpenLocal/resolveLocalPath (by design: OpenLocal is the intended local path entry point)
- CloneOptions.String() redacts Token, Password, SSHKeyPassword but intentionally leaves SSHKeyPath and Username in plaintext
- Subdirectory is prepended with naive string concatenation (no path.Clean, no containment check) — path traversal via "../" in subdirectory or caller-supplied paths is unmitigated
- resolveLocalPath accepts relative paths resolved from CWD — ../escape is possible when the caller controls the path string
- ParseURLSubdir skips scheme "://" before searching for "//" separator — works correctly for known schemes but does no sanitisation of the extracted subdirectory
- go-git v6 is alpha (v6.0.0-alpha.1) — supply chain / stability risk
- cyphar/filepath-securejoin is present as an indirect dependency (pulled in by go-git) but is NOT used directly by this module

**Why this matters:** The module is a shared library used by higher-level CLI commands. Any missing input validation here propagates to all callers.

**How to apply:** When reviewing callers or new features, verify that subdirectory and path inputs are sanitised before reaching this module, or that the module itself gains containment checks.
