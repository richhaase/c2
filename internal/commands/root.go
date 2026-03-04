// Copyright (c) 2026 Rich Haase. All rights reserved.
// Use of this source code is governed by the MIT license.

package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionInfo struct {
	Version string
	Commit  string
	Date    string
}

var rootCmd = &cobra.Command{
	Use:   "c2",
	Short: "Concept2 Logbook CLI",
	Long:  "CLI tool for Concept2 Logbook data sync and analysis.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if v, _ := cmd.Flags().GetBool("version"); v {
			fmt.Printf("c2 %s\n", formatVersion())
			return nil
		}
		return cmd.Help()
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("c2 %s (commit: %s, built: %s)\n",
			versionInfo.Version, versionInfo.Commit, versionInfo.Date)
	},
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "show version information")
	rootCmd.AddCommand(versionCmd)
}

// ExecuteWithExitCode runs the root command and returns the appropriate exit code.
func ExecuteWithExitCode(version, commit, date string) int {
	versionInfo.Version = version
	versionInfo.Commit = commit
	versionInfo.Date = date
	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}

func formatVersion() string {
	if versionInfo.Version == "dev" {
		return fmt.Sprintf("%s-%s", versionInfo.Version, versionInfo.Commit)
	}
	return versionInfo.Version
}
