package build_test

import (
	"github.com/spf13/cobra"

	"github.com/ivanklee86/gantry/cmd/build"
	"github.com/ivanklee86/gantry/pkg/gantry"
)

// newTestRootCommand builds a minimal root command with the build subcommand
// registered, suitable for e2e tests without importing the main package.
func newTestRootCommand() *cobra.Command {
	g := gantry.New()

	root := &cobra.Command{
		Use: "gantry",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			g.Out = cmd.OutOrStdout()
			g.Err = cmd.ErrOrStderr()
			return nil
		},
	}

	root.AddCommand(build.NewBuildCommand(g))
	return root
}
