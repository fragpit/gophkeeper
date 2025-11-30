package main

import (
	"time"

	app "github.com/fragpit/gophkeeper/internal/app/server"
	"github.com/spf13/cobra"
)

func NewRunCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:           "run",
		Short:         "Start server",
		Aliases:       []string{"r"},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Run(cmd.Context(), cfg)
		},
	}

	runCmd.Flags().
		StringP("address", "a", "localhost:8080", "listen address to run server")
	runCmd.Flags().
		StringP("database-uri", "d", "", "database uri, format postgresql://user@localhost:5432")
	runCmd.Flags().
		String("s3-endpoint", "localhost:8333", "S3 compatible object storage endpoint")
	runCmd.Flags().
		String("s3-access-key", "", "S3 compatible object storage access key")
	runCmd.Flags().
		String("s3-secret-key", "", "S3 compatible object storage secret key")
	runCmd.Flags().
		String("tls-cert-file", "", "path to TLS server certificate")
	runCmd.Flags().
		String("tls-key-file", "", "path to TLS server key")
	runCmd.Flags().
		Duration("jwt-ttl", time.Duration(24*time.Hour), "jwt token TTL duration")
	runCmd.Flags().
		String("jwt-secret", "", "server base64 jwt secret")
	runCmd.Flags().
		String("master-key", "", "server base64 master-key")

	return runCmd
}
