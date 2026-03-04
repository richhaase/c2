// Copyright (c) 2026 Rich Haase. All rights reserved.
// Use of this source code is governed by the MIT license.

package main

import (
	"os"
	"runtime/debug"
	"strings"

	"github.com/richhaase/c2cli/internal/commands"
)

// Version information, injected at build time via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// getVersionInfo returns version information, with fallback to build info for go install.
func getVersionInfo() (string, string, string) {
	if version != "dev" {
		return version, commit, date
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		buildVersion := info.Main.Version
		buildCommit := "unknown"
		buildDate := "unknown"

		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if len(setting.Value) > 7 {
					buildCommit = setting.Value[:7]
				} else {
					buildCommit = setting.Value
				}
			case "vcs.time":
				buildDate = setting.Value
			}
		}

		if buildVersion != "" && buildVersion != "(devel)" {
			buildVersion = strings.TrimPrefix(buildVersion, "v")
			return "v" + buildVersion, buildCommit, buildDate
		}
	}

	return version, commit, date
}

func main() {
	v, c, d := getVersionInfo()
	os.Exit(commands.ExecuteWithExitCode(v, c, d))
}
