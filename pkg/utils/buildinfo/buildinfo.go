package buildinfo

import "fmt"

// PrintBuildInfo prints the build version, date, and commit information to standard output.
// If any parameter is empty, it will be displayed as "N/A".
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
