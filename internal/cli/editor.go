package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
)

func editorArguments(value string) ([]string, error) {
	var args []string
	var word strings.Builder
	var quote rune
	escaped, started := false, false
	for _, r := range value {
		switch {
		case escaped:
			word.WriteRune(r)
			escaped = false
		case r == '\\' && quote != '\'':
			escaped, started = true, true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				word.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, started = r, true
		case unicode.IsSpace(r):
			if started {
				args = append(args, word.String())
				word.Reset()
				started = false
			}
		default:
			word.WriteRune(r)
			started = true
		}
	}
	if quote != 0 || escaped {
		return nil, fmt.Errorf("Invalid editor command: unfinished quote or escape. Check VISUAL or EDITOR.")
	}
	if started {
		args = append(args, word.String())
	}
	if len(args) == 0 || args[0] == "" {
		return nil, fmt.Errorf("Editor command is empty. Set VISUAL or EDITOR.")
	}
	return args, nil
}

func editInEditor(cmd *cobra.Command, body string) (string, error) {
	choice := strings.TrimSpace(os.Getenv("VISUAL"))
	if choice == "" {
		choice = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if choice == "" {
		choice = "vi"
	}
	args, err := editorArguments(choice)
	if err != nil {
		return "", err
	}
	file, err := os.CreateTemp("", "c2-note-*.md")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())
	if _, err := file.WriteString(body); err != nil {
		return "", errors.Join(err, file.Close())
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	process := exec.CommandContext(cmd.Context(), args[0], append(args[1:], file.Name())...)
	process.Stdin = cmd.InOrStdin()
	process.Stdout = cmd.ErrOrStderr()
	process.Stderr = cmd.ErrOrStderr()
	if err := process.Run(); err != nil {
		return "", fmt.Errorf("Editor failed; note unchanged. Check VISUAL or EDITOR (GUI editors need a wait option): %w", err)
	}
	data, err := os.ReadFile(file.Name())
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(data)) == "" {
		return "", fmt.Errorf("Editor returned empty text; note unchanged. Empty text does not delete a note.")
	}
	return string(data), nil
}
