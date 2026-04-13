---
name: Gantry Security Architecture
description: Core security architecture, threat model, and known findings for the Gantry devcontainer merge tool
type: project
---

Gantry is a Go CLI tool that clones git repositories into memory and merges devcontainer.json files using go-jsonnet. Key security-relevant facts:

- Input pipeline: user-supplied git URLs + file paths -> `git.Clone` (in-memory) -> `git.GetFiles` -> `merge.Merge` (Jsonnet evaluation)
- `merge.Merge` uses `safePath` regex (`^[a-zA-Z0-9_\-./]+$`) to validate `f.Path` before embedding in Jsonnet snippet, and has defense-in-depth backslash/quote escaping. Injection via path characters is mitigated.
- `MemoryImporter` restricts file access to the pre-loaded in-memory map — Jsonnet VM cannot reach real filesystem or network.
- `MaxStack = 100` is set; `maxOutputBytes = 10 MB` limits output size; no CPU/evaluation-time fuel limit (recursion-via-std-functions could still DoS).
- go-git/v6 is an alpha release (v6.0.0-alpha.1) — supply-chain/stability risk, no stable API guarantee.
- `google/go-jsonnet v0.22.0` is a direct dependency (correct in go.mod).
- `securejoin` (cyphar/filepath-securejoin v0.6.1) used in `GetFiles` to contain file reads to in-memory billy FS root.
- **Known gap**: `Overlay.Subdirectory` from YAML config and `--ssh-key-path`/`--files` from CLI are never validated for `..` components; only `ParseURLSubdir` (URL // syntax) does this check. YAML `subdirectory:` field and direct Overlay struct bypass this protection.
- **Known gap**: `Files []string` in config are not expanded via `expandConfig` — env-var injection in file paths is not possible. However, file paths from YAML are not validated for `..` components; `securejoin` in `GetFiles` is the sole backstop.
- **Known gap**: `--password`, `--token`, `--ssh-key-password` flags appear in plaintext in process argument lists (visible to other OS users via /proc/<pid>/cmdline).
- **Known gap**: `http://` scheme is allowed without credentials, enabling unathenticated MITM on plaintext clone traffic.
- **Known gap**: `git@` (SCP) URLs bypass `validateURL` scheme check entirely; no SSRF/URL-scope restrictions for SSH targets.
- `io.ReadAll` on in-memory billy FS files has no per-file size limit; a malicious repo could create a very large file that exhausts memory before `maxOutputBytes` check.
- Viper config auto-loading (`gantry.yaml` in CWD) can silently override CLI flags including credential flags.
- Error messages containing user-controlled overlay repo names are passed through `stripansi.Strip` (ANSI-safe) but not otherwise sanitized for log injection.

- **New (wire_up_cli CLI layer)**: `main.go` wires Viper with `AutomaticEnv()` and `bindFlags`. Credential flags (`--token`, `--password`, `--ssh-key-password`) are mapped to env vars (`GANTRY_TOKEN`, `GANTRY_PASSWORD`, `GANTRY_SSH_KEY_PASSWORD`) via `BindEnv`, so they can be passed via env rather than process args — this is an improvement over the prior known gap.
- **New (wire_up_cli CLI layer)**: `bindFlags` only maps flags whose names contain `-` (hyphen) to env vars. Flags without hyphens (e.g., `--token`, `--username`) are NOT auto-mapped to GANTRY_TOKEN / GANTRY_USERNAME unless Viper's `AutomaticEnv()` picks them up with `GANTRY_` prefix. Verify actual effective env var names.
- **New (wire_up_cli CLI layer)**: `writeOutput` uses `0o644` permissions for the output file — world-readable. If output_path is in a shared directory, merged devcontainer.json (which may embed credentials via env-var expansion) is readable by all local users.
- **New (wire_up_cli CLI layer)**: `MkdirAll` uses `0o755` — created directories are world-readable/traversable.
- **New**: SSH host key verification: `ssh.NewPublicKeysFromFile` and `ssh.NewSSHAgentAuth` in go-git v6-alpha do not appear to enforce strict host key checking by default. No `HostKeyCallback` or `knownhosts` usage found. This means SSH clones are susceptible to MITM.
- **Known gap (confirmed still present)**: `Overlay.Subdirectory` from YAML/CLI not validated for `..`; securejoin is sole backstop.
- **Known gap (confirmed still present)**: `http://` scheme allowed without credentials; unauthenticated plaintext clone possible.
- **Known gap (confirmed still present)**: `git@` SCP URLs bypass `validateURL` entirely.
- **Known gap (confirmed still present)**: No per-file read size limit in `io.ReadAll` within `GetFiles`.
- **Known gap (confirmed still present)**: Viper auto-loads `gantry.yaml` from CWD silently.

**Why:** Comprehensive security review of the full wire_up_cli branch codebase, updated after CLI layer review.
**How to apply:** Use as baseline for all future reviews of merge.go, git.go, config.go, build.go, and main.go/cmd/build/build.go.
