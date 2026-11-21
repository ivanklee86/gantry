{
  name: "gantry",
  customizations+: {
    vscode+: {
      extensions+: [
        "DavidAnson.vscode-markdownlint",
      ],
    },
  },
  postCreateCommand+: {
    "gantry-post-install": "bash ./.devcontainer/post_install.sh",
  },
}
