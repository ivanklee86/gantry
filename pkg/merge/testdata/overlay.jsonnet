{
    customizations+: {
        vscode+: {
            extensions+: [
                "charliermarsh.ruff",
                "astral-sh.ty"
            ],
            settings+: {
                "python.linting.enabled": std.length("enabled") > 0
            }
        }
    },
    postCreateCommand+: {
        "install-prek": "prek install && prek run"
    }
}
