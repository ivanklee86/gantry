// team-override.jsonnet
//
// Team-level overlay applied on top of the shared Go base devcontainer.
//
// Each key listed here REPLACES the corresponding key from the base.
// Keys not listed are inherited unchanged from the base.
//
// Note: Jsonnet's + operator performs a shallow merge on objects. Array values
// (such as extensions) are fully replaced — to append to the base list, copy
// the base entries here and add your own.
{
  // Give this workspace a team-specific name.
  name: "myteam-go",

  // Switch from a Dockerfile build to a pre-built team image.
  // Setting build to null removes it from the merged output.
  build: null,
  image: "ghcr.io/myteam/devcontainer/go:1.26",

  customizations: {
    vscode: {
      settings: {
        // Team-wide VS Code settings (replaces the base settings block).
        "yaml.schemas": {
          "https://taskfile.dev/schema.json": ["**/Taskfile.yml", "tasks/**"],
        },
        "editor.formatOnSave": true,
        "editor.rulers": [100],
      },
      // Full extensions list: base extensions + team additions.
      extensions: [
        "anthropic.claude-code",
        "GitHub.copilot",
        "golang.go",
        "ms-azuretools.vscode-docker",
        "redhat.vscode-yaml",
        "tamasfe.even-better-toml",
        "task.vscode-task",
        // Team-specific additions.
        "eamodio.gitlens",
        "ms-vscode.makefile-tools",
      ],
    },
  },

  // Add a team-specific environment variable.
  remoteEnv: {
    TEAM: "platform",
  },
}
