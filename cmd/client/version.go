package main

import (
	"github.com/fragpit/gophkeeper/pkg/utils/buildinfo"
	"github.com/spf13/cobra"
)

var buildVersion string
var buildDate string
var buildCommit string

func NewVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"v", "ver"},
		Short:   "Show app version",
		Run: func(cmd *cobra.Command, args []string) {
			buildinfo.PrintBuildInfo(buildVersion, buildDate, buildCommit)
		},
	}

	return versionCmd
}
