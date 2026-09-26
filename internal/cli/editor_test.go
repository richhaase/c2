package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestEditorArguments(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  []string
	}{
		{`code --wait`, []string{"code", "--wait"}},
		{`'/Applications/My Editor' --wait "two words"`, []string{"/Applications/My Editor", "--wait", "two words"}},
		{`my\ editor --flag=''`, []string{"my editor", "--flag="}},
		{`editor '$(touch nope)' ';'`, []string{"editor", "$(touch nope)", ";"}},
	} {
		got, err := editorArguments(tc.input)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%q = %v, %v", tc.input, got, err)
		}
	}
	for _, value := range []string{"", "''", "editor 'unfinished", "editor \\"} {
		if _, err := editorArguments(value); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
}

func TestEditorProcess(t *testing.T) {
	if os.Getenv("C2_TEST_EDITOR") != "1" {
		return
	}
	path := os.Args[len(os.Args)-1]
	root, err := os.OpenRoot(os.TempDir())
	if err != nil {
		os.Exit(9)
	}
	name := filepath.Base(path)
	data, err := root.ReadFile(name)
	if err != nil || string(data) != "original\n" {
		os.Exit(10)
	}
	info, err := root.Stat(name)
	if err != nil || info.Mode().Perm() != 0o600 {
		os.Exit(11)
	}
	fmt.Fprintln(os.Stderr, path)
	switch os.Args[len(os.Args)-2] {
	case "fail":
		os.Exit(12)
	case "empty":
		err = root.WriteFile(name, []byte(" \n"), 0o600)
	case "replace":
		err = root.WriteFile(name, []byte("  corrected < & >\n\n"), 0o600)
	}
	if err != nil {
		os.Exit(13)
	}
	if err := root.Close(); err != nil {
		os.Exit(14)
	}
	os.Exit(0)
}

func TestEditorLifecycle(t *testing.T) {
	for _, mode := range []string{"replace", "unchanged", "empty", "fail"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("C2_TEST_EDITOR", "1")
			command := fmt.Sprintf("%q -test.run=^TestEditorProcess$ -- %s", os.Args[0], mode)
			t.Setenv("EDITOR", "missing-editor")
			t.Setenv("VISUAL", command)
			if mode == "unchanged" {
				t.Setenv("VISUAL", "")
				t.Setenv("EDITOR", command)
			}
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())
			var out, diagnostics bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&diagnostics)
			cmd.SetIn(strings.NewReader(""))
			got, err := editInEditor(cmd, "original\n")
			if (err != nil) != (mode == "fail" || mode == "empty") {
				t.Fatalf("result %q, %v", got, err)
			}
			if mode == "replace" && got != "  corrected < & >\n\n" || mode == "unchanged" && got != "original\n" {
				t.Fatalf("text changed: %q", got)
			}
			if out.Len() != 0 {
				t.Fatal("editor contaminated stdout")
			}
			path := strings.TrimSpace(diagnostics.String())
			if path == "" {
				t.Fatal("editor did not run")
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("temporary file retained: %v", err)
			}
		})
	}
}
