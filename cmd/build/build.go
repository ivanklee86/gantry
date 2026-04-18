package build

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ivanklee86/gantry/pkg/config"
	"github.com/ivanklee86/gantry/pkg/gantry"
)

// NewBuildCommand returns the `gantry build` Cobra subcommand.
// g must be non-nil; its Out/Err writers are set by the root command's PersistentPreRunE.
func NewBuildCommand(g *gantry.Gantry) *cobra.Command {
	var (
		configFile     string
		repo           string
		ref            string
		commit         string
		files          []string
		outputPath     string
		write          bool
		token          string
		username       string
		password       string
		sshKeyPath     string
		sshKeyPassword string
	)

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a devcontainer.json from one or more repository overlays.",
		Long: `Build merges devcontainer.json files sourced from one or more git repositories
and writes the result to stdout or to a file.

Pass --config to use a YAML configuration file (supports multiple repositories).
Pass --repo and --files for a quick single-repository build on the command line.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			var cfg config.BuildConfig
			var err error

			if configFile != "" {
				cfg, err = config.LoadYAMLConfig(configFile)
				if err != nil {
					g.Error(err)
					return err
				}
				// CLI flags override YAML values when explicitly set.
				if outputPath != "" {
					cfg.OutputPath = outputPath
				}
				if write {
					cfg.Write = true
				}
			} else {
				cfg, err = config.CLIFlagsToConfig(config.CLIFlags{
					Repo:           repo,
					Ref:            ref,
					Commit:         commit,
					Files:          files,
					OutputPath:     outputPath,
					Write:          write,
					Token:          token,
					Username:       username,
					Password:       password,
					SSHKeyPath:     sshKeyPath,
					SSHKeyPassword: sshKeyPassword,
				})
				if err != nil {
					g.Error(err)
					return err
				}
			}

			if err := g.Build(ctx, cfg); err != nil {
				g.Error(err)
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "", "Path to a YAML configuration file.")

	cmd.Flags().StringVar(&repo, "repo", "", "Repository URL or local path. Append //subdir to set a subdirectory.")
	cmd.Flags().StringVar(&ref, "ref", "", "Branch or tag reference (e.g. refs/heads/main). Mutually exclusive with --commit.")
	cmd.Flags().StringVar(&commit, "commit", "", "Exact commit SHA to check out (7-40 hex characters). Mutually exclusive with --ref.")
	cmd.Flags().StringArrayVar(&files, "files", nil, "File path to extract (repeatable, applied in order).")
	cmd.Flags().StringVar(&outputPath, "output-path", "", "Destination path for the merged output.")
	cmd.Flags().BoolVar(&write, "write", false, "Write output to --output-path instead of stdout.")

	cmd.Flags().StringVar(&token, "token", "", "Bearer token for HTTPS authentication.")
	cmd.Flags().StringVar(&username, "username", "", "Username for HTTPS basic auth.")
	cmd.Flags().StringVar(&password, "password", "", "Password for HTTPS basic auth.")
	cmd.Flags().StringVar(&sshKeyPath, "ssh-key-path", "", "Path to SSH private key file.")
	cmd.Flags().StringVar(&sshKeyPassword, "ssh-key-password", "", "Passphrase for an encrypted SSH private key.")

	return cmd
}
