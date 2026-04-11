# Gantry

Gantry is a CLI tool that makes setting up `devcontainer.json` quick, easy, and painless for large teams and microservices!

It should:
- Be a Go CLI using Cobra/Viper.
- Inputs:
  - Git repository (`--git-repo`)
  - Reference a series of files (`--files`)
  - Flag to enable writes `--write`
- Use `git-go` to clone repository into memory and extract files specified.
- Use `go-jsonnet` to merge the files in specified order.
- Write to `devcontainer.json` if `--write` is specified.

Notes:
- Logging for humans should be handled in the `gantry` package itself.
