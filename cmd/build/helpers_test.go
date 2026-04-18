package build_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/ivanklee86/gantry/cmd/build"
	"github.com/ivanklee86/gantry/pkg/gantry"
)

// repoRoot is the absolute path to the repository root, determined once at init time.
var repoRoot string

func init() {
	// Walk up from the test binary's working directory until we find go.mod.
	dir, err := os.Getwd()
	if err != nil {
		panic("cannot determine working directory: " + err.Error())
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			repoRoot = dir
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("cannot find repository root (no go.mod found)")
		}
		dir = parent
	}
}

// exampleConfig reads an example config file, replaces the portable "./" repo
// placeholder with the absolute repository root, writes the result to a temp
// file, and returns its path. This lets example configs use "./" for
// human-friendly manual use while still working correctly in tests.
func exampleConfig(t *testing.T, relPath string) string {
	t.Helper()
	src := filepath.Join(repoRoot, relPath)
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	updated := strings.ReplaceAll(string(data), `repo: "./"`, `repo: "`+repoRoot+`"`)
	dir := t.TempDir()
	dst := filepath.Join(dir, filepath.Base(src))
	require.NoError(t, os.WriteFile(dst, []byte(updated), 0o600))
	return dst
}

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
