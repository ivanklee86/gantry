package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"

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
	gantry := gantry.New()

	cmd := &cobra.Command{
		Use:     "gantry",
		Short:   "Enforce minimum versions in CI/CD.",
		Long:    "A CLI to enforce minimum versions for packages in CI/CD.",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			gantry.Out = cmd.OutOrStdout()
			gantry.Err = cmd.ErrOrStderr()

			return initializeConfig(cmd)
		},
		Run: func(cmd *cobra.Command, args []string) {
			_, err := fmt.Fprint(gantry.Out, cmd.UsageString())
			if err != nil {
				gantry.Error(err)
			}
		},
	}

	cmd.PersistentFlags().BoolVar(&gantry.NoExitCode, "no-exit-on-fail", false, "Don't return a non-zero exit code on failure.")

	return cmd
}

func initializeConfig(cmd *cobra.Command) error {
	v := viper.New()

	v.SetConfigName(defaultConfigFilename)
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	bindFlags(cmd, v)

	return nil
}

func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	processed := make(map[string]struct{})

	processFlagSet := func(fs *pflag.FlagSet) {
		if fs == nil {
			return
		}

		fs.VisitAll(func(f *pflag.Flag) {
			// Avoid processing the same flag multiple times if it appears in more than one FlagSet.
			if _, ok := processed[f.Name]; ok {
				return
			}
			processed[f.Name] = struct{}{}

			if strings.Contains(f.Name, "-") {
				envVarSuffix := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
				if err := v.BindEnv(f.Name, fmt.Sprintf("%s_%s", envPrefix, envVarSuffix)); err != nil {
					os.Exit(1)
				}
			}

			if !f.Changed && v.IsSet(f.Name) {
				val := v.Get(f.Name)
				if err := fs.Set(f.Name, fmt.Sprintf("%v", val)); err != nil {
					os.Exit(1)
				}
			}
		})
	}

	processFlagSet(cmd.Flags())
	processFlagSet(cmd.PersistentFlags())
	processFlagSet(cmd.InheritedFlags())
}
