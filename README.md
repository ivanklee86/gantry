# gantry

A CLI for Platform-izing devcontainers in microservices, widely distributed, or just complicated™ situations.

## Why gantry?

[Devcontainers](https://containers.dev/) are an amazing tool for building shared, secure development environments for distributed teams.  However, managing the configuration (Dockerfile, `devcontainer.json`) comes with its own complexities: teams want to customize tools, programming languages have different standards, and services proliferate.  

There are well-known solutions for [Dockerfiles](https://github.com/ivanklee86/devcontainers) like using base images and [docker bake](https://docs.docker.com/build/bake/reference/).  

But what if you're a development team that wants a standard VSCode configuration for the [astral](https://astral.sh) stack?  Or a Platform Engineering team standardizing your AI coding tool stack?  

`gantry` is a light-weight CLI that solves this problem by:
  
  - Allowing teams to store `devcontainer.json` configuration in git repositories.
  - Uses [jsonnet](https://jsonnet.org/) to merge standard configurations (e.g. a base Python `devcontainer.json`) with use-case (e.g. a Jupyter plugin for data teams) and/or team (e.g. Svelte for a fullstack team) from remote repositories.
  - Let developers define a configuration as a YAML file on disk and re-generate the `devcontainer.json` on demand.

## Installing gantry

### With go

```sh
go install github.com/ivanklee86/gantry@latest
```

### With Homebrew

```sh
brew install ivanklee86/tap/gantry
```

### With Docker

```sh
docker pull ghcr.io/ivanklee86/gantry:latest
```

To embed `gantry` in your own Dockerfile:

```dockerfile
COPY --from=ghcr.io/ivanklee86/gantry:latest /usr/local/bin/gantry /usr/local/bin/gantry
```

## Usage and Configuration

### Quick start (CLI flags)

Build from a single remote repository and print to stdout:

```sh
gantry build --repo https://github.com/org/devcontainers --ref refs/heads/main --files base.jsonnet --files python.jsonnet
```

Write the result to a file:

```sh
gantry build --repo https://github.com/org/devcontainers --ref refs/heads/main \
  --files base.jsonnet --files python.jsonnet \
  --write --output-path .devcontainer/devcontainer.json
```

Pin to an exact commit:

```sh
gantry build --repo https://github.com/org/devcontainers --commit abc1234 --files base.jsonnet
```

### YAML configuration file

For multi-repository merges or to keep the configuration in source control, use a YAML file:

```yaml
version: 1

output_path: .devcontainer/devcontainer.json

overlays:
  - repo: https://github.com/org/devcontainers
    ref: refs/heads/main
    files:
      - base.jsonnet
      - python.jsonnet
  - repo: https://github.com/org/team-overlays
    ref: refs/heads/main
    subdirectory: data-team
    files:
      - jupyter.jsonnet
```

Run it:

```sh
gantry build --config gantry.yaml --write
```

CLI flags `--output-path` and `--write` override the values in the config file when specified.

### Multi-layer example

A typical setup has three layers: a shared org base, a language overlay, and a team overlay. Each layer is a Jsonnet file that merges on top of the previous result.

**Layer 1 — org base** (`devcontainers/bases/base.json`, plain JSON):
```json
{
  "name": "base",
  "image": "mcr.microsoft.com/devcontainers/base:ubuntu-24.04",
  "remoteUser": "vscode",
  "customizations": {
    "vscode": {
      "settings": {
        "editor.formatOnSave": true,
        "editor.rulers": [100]
      },
      "extensions": ["GitHub.vscode-github-actions", "redhat.vscode-yaml"]
    }
  },
  "postCreateCommand": {
    "upgrade": "sudo apt-get update && sudo apt-get upgrade -y"
  }
}
```

**Layer 2 — language overlay** (`devcontainers/languages/python.jsonnet`):

Uses `+:` to deep-merge into the base rather than replacing it.

```jsonnet
{
  name: "python",
  image: "mcr.microsoft.com/devcontainers/python:3.13",

  customizations+: {
    vscode+: {
      settings+: {
        "python.defaultInterpreterPath": "/usr/local/bin/python",
        "[python]": { "editor.defaultFormatter": "charliermarsh.ruff" },
      },
      extensions+: [
        "charliermarsh.ruff",
        "ms-python.python",
        "ms-python.mypy-type-checker",
      ],
    },
  },

  postCreateCommand+: {
    "install-uv": "curl -LsSf https://astral.sh/uv/install.sh | sh",
  },
}
```

**Layer 3 — team overlay** (`team-overlays/data-team/jupyter.jsonnet`):

```jsonnet
{
  name: "data-team-python",

  customizations+: {
    vscode+: {
      extensions+: [
        "ms-toolsai.jupyter",
        "ms-toolsai.vscode-jupyter-cell-tags",
      ],
    },
  },

  remoteEnv: {
    TEAM: "data",
    JUPYTER_PORT: "8888",
  },

  forwardPorts: [8888],

  postCreateCommand+: {
    "install-deps": "uv sync --all-extras",
  },
}
```

**`gantry.yaml`** wiring all three together:

```yaml
version: 1

output_path: .devcontainer/devcontainer.json

overlays:
  - repo: https://github.com/org/devcontainers
    ref: refs/heads/main
    subdirectory: bases
    files:
      - base.json
  - repo: https://github.com/org/devcontainers
    ref: refs/heads/main
    subdirectory: languages
    files:
      - python.jsonnet
  - repo: https://github.com/org/team-overlays
    ref: refs/heads/main
    subdirectory: data-team
    files:
      - jupyter.jsonnet
```

```sh
gantry build --config gantry.yaml --write
```

### Authentication

| Method | Flags |
|---|---|
| Token (HTTPS) | `--token` |
| Basic auth | `--username` / `--password` |
| SSH key | `--ssh-key-path` / `--ssh-key-password` |

In YAML configs, all string fields support `${ENV_VAR}` interpolation:

```yaml
overlays:
  - repo: https://github.com/org/private-devcontainers
    ref: refs/heads/main
    files:
      - base.jsonnet
    auth:
      token: ${GITHUB_TOKEN}
```
