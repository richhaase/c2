package main

import (
	"strings"
	"testing"
)

func TestBuildVersionStringIncludesName(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	t.Cleanup(func() {
		version, commit, date = oldVersion, oldCommit, oldDate
	})

	version = "v1.2.3"
	commit = "abc1234"
	date = "2026-04-26T00:00:00Z"

	got := buildVersionString()
	for _, want := range []string{"c2", "v1.2.3", "abc1234", "2026-04-26T00:00:00Z"} {
		if !strings.Contains(got, want) {
			t.Fatalf("buildVersionString() = %q, missing %q", got, want)
		}
	}
}
