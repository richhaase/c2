package main

import (
	"fmt"
	"runtime/debug"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func buildVersionString() string {
	ver, rev, built := getVersionInfo()
	return fmt.Sprintf("c2 %s (commit: %s, built: %s)", ver, rev, built)
}

func getVersionInfo() (ver, rev, built string) {
	ver, rev, built = version, commit, date
	if ver != "dev" {
		return ver, rev, built
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			ver = info.Main.Version
		}
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if len(setting.Value) >= 7 {
					rev = setting.Value[:7]
				} else if setting.Value != "" {
					rev = setting.Value
				}
			case "vcs.time":
				if setting.Value != "" {
					built = setting.Value
				}
			case "vcs.modified":
				if setting.Value == "true" && rev != "none" {
					rev += "-dirty"
				}
			}
		}
	}
	return ver, rev, built
}
