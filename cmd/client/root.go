package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/fragpit/gophkeeper/cmd/client/config"
	"github.com/fragpit/gophkeeper/internal/app/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type CLI struct {
	Cfg    *config.ClientConfig
	Client *client.Client
}

var (
	debug bool
	cli   CLI
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
		Use:   "gophkeeper-cli",
		Short: "GophKeeper client",
		Long: `
GophKeeper is a client-server system that allows users to reliably and securely
store logins, passwords, binary data, and other private information.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			debug = viper.GetBool("debug")
			cfgFile := viper.GetString("config_file")
			initViperConfig(cfgFile)

			cfg, err := config.NewClientConfig()
			if err != nil {
				return fmt.Errorf("init config: %w", err)
			}
			cli.Cfg = cfg

			if debug {
				handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
					Level: slog.LevelDebug,
				})
				slog.SetDefault(slog.New(handler))
				slog.Debug("debug mode enabled")

				sc := &SafeConfig{
					viper: viper.GetViper(),
					secretKeys: map[string]struct{}{
						"password": struct{}{},
						"token":    struct{}{},
					},
				}
				sc.DumpForDebug()
			} else {
				handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
					Level: slog.LevelInfo,
				})
				slog.SetDefault(slog.New(handler))
			}

			cl, err := client.NewClient(
				cli.Cfg.ServerURL,
				viper.GetString("auth-config"),
			)
			if err != nil {
				return err
			}
			cli.Client = cl

			return nil
		},
	}

	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	rootCmd.PersistentFlags().
		StringP("config-file", "c", "", "path to config file")
	rootCmd.PersistentFlags().
		StringP("server-url", "s", "", "enable debug logging")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get user home dir: %w", err)
	}
	rootCmd.Flags().
		String("auth-config", homeDir+"/.gophkeeper/"+"auth.cfg", "file name for saving auth data")
	if err := viper.BindPFlag("auth-config", rootCmd.Flags().Lookup("auth-config")); err != nil {
		return nil, fmt.Errorf("bind viper flag: %w", err)
	}

	rootCmd.AddCommand(NewVersionCmd())
	rootCmd.AddCommand(NewCreateCmd())
	rootCmd.AddCommand(NewGetCmd())
	rootCmd.AddCommand(NewLoginCmd())
	rootCmd.AddCommand(NewListCmd())

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
		viper.SetConfigName(".gophkeeper-cli")
	}

	viper.SetEnvPrefix("GK")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func bindFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		normalized := strings.ReplaceAll(f.Name, "-", "_")
		_ = viper.BindPFlag(normalized, f)
	})

	for _, c := range cmd.Commands() {
		bindFlags(c)
	}
}
