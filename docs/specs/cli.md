# Gantry

Gantry is a CLI tool that makes setting up `devcontainer.json` quick, easy, and painless for large teams and microservices!

It should:
- Be a Go CLI using Cobra/Viper.
- Use `git-go` to clone repository into memory and extract files specified.
- Use `go-jsonnet` to merge the files in specified order.
- Write to `devcontainer.json` if `--write` is specified.

Notes:
- Logging for humans should be handled in the `gantry` package itself.

## Commands

### gantry build

Command should take inputs in two formats:

#### Command line arguments

Single-repo use case. All auth flags are optional.

| Flag | Description |
|---|---|
| `--repo` | Repository URL or local path. Remote URLs support `https://`, `http://`, `ssh://`, `git://`, and `git@` (SCP-style). Local paths may be absolute, relative (`./`, `../`), or `file://` URIs. Append `//subdir` to set a subdirectory (e.g. `https://github.com/org/repo//configs`). |
| `--ref` | Branch or tag reference (e.g. `refs/heads/main`, `refs/tags/v1.2.0`). Uses remote default branch if omitted. Mutually exclusive with `--commit`. |
| `--commit` | Exact commit SHA to check out (7–40 hex characters). Mutually exclusive with `--ref`. |
| `--files` | One or more file paths to extract from the repository, in overlay order. Repeatable (e.g. `--files base.jsonnet --files override.jsonnet`). |
| `--output-path` | Destination path for the merged output. |
| `--write` | Write output to `--output-path`. Without this flag, output is printed to stdout. |
| `--token` | Bearer token for HTTPS authentication. |
| `--username` | Username for HTTPS basic auth. |
| `--password` | Password for HTTPS basic auth. |
| `--ssh-key-path` | Path to SSH private key file. If omitted, the SSH agent is used. |
| `--ssh-key-password` | Passphrase for an encrypted SSH private key. |

#### YAML file (`--config`)

Pass `--config <path>` to read configuration from a YAML file. Supports multiple repositories and overlays in merge order.

```yaml
version: 1
output_path: "path/to/devcontainer.json"

overlays:
  # Remote repository — HTTPS with token from environment variable
  - repo: "https://github.com/org/devcontainers"
    ref: "refs/heads/main"          # optional; branch or tag; mutually exclusive with commit
    # commit: "abc1234"             # optional; 7-40 hex char SHA; mutually exclusive with ref
    subdirectory: "configs/base"    # optional; equivalent to the // URL syntax
    files:
      - "devcontainer.json"
    auth:                           # optional
      token: "${GITHUB_TOKEN}"      # supports ${ENV_VAR} interpolation

  # Remote repository — SSH with key file
  - repo: "git@github.com:org/overlays.git"
    ref: "refs/heads/stable"
    files:
      - "python.jsonnet"
    auth:
      ssh_key_path: "~/.ssh/id_rsa_deploy"
      ssh_key_password: "${SSH_KEY_PASS}"   # optional

  # Remote repository — HTTPS basic auth
  - repo: "https://internal.example.com/configs.git"
    files:
      - "team-defaults.jsonnet"
    auth:
      username: "${GIT_USER}"
      password: "${GIT_PASS}"

  # Local repository — useful for testing or co-located configs
  - repo: "./local-configs"         # absolute, relative, or file:// URI
    files:
      - "local-override.jsonnet"
```

Auth field reference for an overlay:

| Field | Description |
|---|---|
| `token` | Bearer token for HTTPS. Takes precedence over `username`/`password`. |
| `username` | Username for HTTPS basic auth. |
| `password` | Password for HTTPS basic auth. |
| `ssh_key_path` | Path to SSH private key. If omitted, uses the SSH agent. |
| `ssh_key_password` | Passphrase for an encrypted SSH private key. |

Notes:
- `${ENV_VAR}` references in any string field are expanded at runtime. This keeps credentials out of config files.
- `ref` and `commit` are mutually exclusive within an overlay.
- Files within each overlay are merged in the order listed. Overlays themselves are applied in the order listed.
- Local paths (`./`, `../`, `/`, `file://`) are opened with `OpenLocal`; all other values are treated as remote URLs and cloned with `Clone`.
