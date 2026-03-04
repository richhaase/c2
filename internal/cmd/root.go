// Copyright (c) 2026 Rich Haase. All rights reserved.
// Use of this source code is governed by the MIT license.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRootCmd(version, commit, date string) *cobra.Command {
	root := &cobra.Command{
		Use:           "c2cli",
		Short:         "Concept2 Logbook CLI",
		Long:          "CLI tool for Concept2 Logbook data sync and analysis.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newAuthCmd())
	root.AddCommand(newSyncCmd())
	root.AddCommand(newLogCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newTrendCmd())
	root.AddCommand(newExportCmd())
	root.AddCommand(newVersionCmd(version, commit, date))

	return root
}

func newVersionCmd(version, commit, date string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("c2cli %s (commit: %s, built: %s)\n", version, commit, date)
		},
	}
}
