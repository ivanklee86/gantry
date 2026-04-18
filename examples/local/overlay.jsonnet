// overlay.jsonnet
//
// Team overlay applied on top of base.json.
//
// Uses +: (hidden concatenation) to deep-merge nested objects and arrays
// rather than replacing them wholesale. Fields not listed here are inherited
// unchanged from the base.
{
  // Give this workspace a team-specific name.
  name: "my-project-team",

  customizations+: {
    vscode+: {
      settings+: {
        "editor.rulers": [100, 120],
      },
      // Append team extensions to the base list.
      extensions+: [
        "eamodio.gitlens",
        "redhat.vscode-yaml",
      ],
    },
  },

  postCreateCommand+: {
    "install-tools": "pip install pre-commit && pre-commit install",
  },
}
