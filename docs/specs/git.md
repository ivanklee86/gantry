# git

## Repository sources

We should support the following methods of referencing a repository:

- **HTTPS** — `https://github.com/user/repo.git`
- **SSH** — `git@github.com:user/repo.git`
- **Git protocol** — `git://github.com/user/repo.git` (unauthenticated, read-only)
- **Local absolute path** — `/path/to/repo`
- **Local relative path** — `./relative/path` (resolved relative to CWD)
- **File URI** — `file:///path/to/repo`

## Ref pinning

Repositories may be referenced at a specific point in history. We should support:

- **Branch** — e.g. `main`, `feature/foo`
- **Tag** — e.g. `v1.2.0`
- **Commit SHA** — full or short (e.g. `abc1234`)

When no ref is specified, the default branch (`HEAD`) should be used.

## Subdirectory references

It should be possible to reference a specific directory within a repository rather than the root. For example:

- `https://github.com/user/repo//subdir`
- `/path/to/local/repo//subdir`

The `//` separator (borrowed from Terraform/Kustomize convention) distinguishes the repo root from the subdirectory path.

## Authentication

For HTTPS remotes:

- **Token-based** — GitHub PAT, GitLab token, etc., passed via environment variable or credential helper
- **`~/.netrc`** — standard netrc file for credential storage
- **Git credential helpers** — defer to the system git credential store

For SSH remotes:

- **SSH agent** — use the running `ssh-agent` (default)
- **Key file** — explicit path to a private key file
