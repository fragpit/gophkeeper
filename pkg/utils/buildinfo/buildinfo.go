package buildinfo

import "fmt"

func PrintBuildInfo(version, date, commit string) {
	if version == "" {
		version = "N/A"
	}

	if date == "" {
		date = "N/A"
	}

	if commit == "" {
		commit = "N/A"
	}

	fmt.Printf("Build version: %s\n", version)
	fmt.Printf("Build date: %s\n", date)
	fmt.Printf("Build commit: %s\n", commit)
}
