package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"github.com/ivanklee86/gantry/cmd/build"
	"github.com/ivanklee86/gantry/pkg/gantry"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Build information (injected by goreleaser).
	version = "dev"
)

const (
	defaultConfigFilename = "gantry"
	envPrefix             = "GANTRY"
)

// main function.
func main() {
	command := NewRootCommand()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}

func NewRootCommand() *cobra.Command {
	g := gantry.New()
	var v *viper.Viper

	cmd := &cobra.Command{
		Use:           "gantry",
		Short:         "Enforce minimum versions in CI/CD.",
		Long:          "A CLI to enforce minimum versions for packages in CI/CD.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			g.Out = cmd.OutOrStdout()
			g.Err = cmd.ErrOrStderr()

			var err error
			v, err = initializeConfig(cmd)
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprint(g.Out, cmd.UsageString())
			return err
		},
	}

	cmd.AddCommand(build.NewBuildCommand(g))

	_ = v // available to subcommand closures added via cmd.AddCommand

	return cmd
}

func initializeConfig(cmd *cobra.Command) (*viper.Viper, error) {
	v := viper.New()

	v.SetConfigName(defaultConfigFilename)
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()

	if err := bindFlags(cmd, v); err != nil {
		return nil, err
	}

	return v, nil
}

func bindFlags(cmd *cobra.Command, v *viper.Viper) error {
	var bindErr error
	processed := make(map[string]struct{})

	processFlagSet := func(fs *pflag.FlagSet) {
		if fs == nil || bindErr != nil {
			return
		}

		fs.VisitAll(func(f *pflag.Flag) {
			if bindErr != nil {
				return
			}

			// Avoid processing the same flag multiple times if it appears in more than one FlagSet.
			if _, ok := processed[f.Name]; ok {
				return
			}
			processed[f.Name] = struct{}{}

			if strings.Contains(f.Name, "-") {
				envVarSuffix := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
				if err := v.BindEnv(f.Name, fmt.Sprintf("%s_%s", envPrefix, envVarSuffix)); err != nil {
					bindErr = err
					return
				}
			}

			if !f.Changed && v.IsSet(f.Name) {
				val := v.Get(f.Name)
				if err := fs.Set(f.Name, fmt.Sprintf("%v", val)); err != nil {
					bindErr = err
				}
			}
		})
	}

	processFlagSet(cmd.Flags())
	processFlagSet(cmd.PersistentFlags())
	processFlagSet(cmd.InheritedFlags())

	return bindErr
}
