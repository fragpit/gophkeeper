package main

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/fragpit/gophkeeper/cmd/server/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	cfg   *config.ServerConfig
	debug bool
)

type SafeConfig struct {
	viper      *viper.Viper
	secretKeys map[string]struct{}
}

func (c *SafeConfig) DumpForDebug() {
	slog.Debug("dump configuration")

	for key, value := range c.viper.AllSettings() {
		if _, secret := c.secretKeys[key]; secret {
			slog.Debug("dump parameter", "name", key, "value", "***")
		} else {
			slog.Debug("dump parameter", "name", key, "value", value)
		}
	}
}

func NewRootCmd() (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:   "gophkeeper",
		Short: "GophKeeper server",
		Long: `
GophKeeper is a client-server system that allows users to reliably and securely
store logins, passwords, binary data, and other private information.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			debug = viper.GetBool("debug")
			if debug {
				logLevel.Set(slog.LevelDebug)
			} else {
				logLevel.Set(slog.LevelInfo)
			}
			cfgFile := viper.GetString("config_file")
			initViperConfig(cfgFile)

			var err error
			cfg, err = config.NewServerConfig()
			if err != nil {
				return fmt.Errorf("init config: %w", err)
			}

			if debug {
				slog.Debug("debug mode enabled")

				sc := &SafeConfig{
					viper: viper.GetViper(),
					secretKeys: map[string]struct{}{
						"password":      struct{}{},
						"jwt_secret":    struct{}{},
						"master_key":    struct{}{},
						"s3_access_key": struct{}{},
						"s3_secret_key": struct{}{},
					},
				}
				sc.DumpForDebug()
			}

			return nil
		},
	}

	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	rootCmd.PersistentFlags().
		StringP("config-file", "c", "", "path to config file")

	rootCmd.AddCommand(NewVersionCmd())
	rootCmd.AddCommand(NewRunCmd())

	bindFlags(rootCmd)

	return rootCmd, nil
}

// initViperConfig reads in config file and ENV variables if set.
func initViperConfig(cfgFile string) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".gophkeeper")
	}

	viper.SetEnvPrefix("GK")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		slog.Info("Using config file", "path", viper.ConfigFileUsed())
	}
}

func bindFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		normalized := strings.ReplaceAll(f.Name, "-", "_")
		_ = viper.BindPFlag(normalized, f)
	})

	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		normalized := strings.ReplaceAll(f.Name, "-", "_")
		_ = viper.BindPFlag(normalized, f)
	})

	for _, c := range cmd.Commands() {
		bindFlags(c)
	}
}
