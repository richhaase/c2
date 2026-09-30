package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type contentInput struct {
	file string
	body string
}

func (c *contentInput) flags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.file, "file", "", "read replacement text from a file, or '-' for stdin")
	cmd.Flags().StringVar(&c.body, "body", "", "replacement text supplied directly")
	cmd.MarkFlagsMutuallyExclusive("file", "body")
}

func (c contentInput) supplied(cmd *cobra.Command, args []string) bool {
	return len(args) > 0 || cmd.Flags().Changed("file") || cmd.Flags().Changed("body")
}

func (c contentInput) read(cmd *cobra.Command, args []string, positionalFile bool) (string, error) {
	if len(args) > 0 && (cmd.Flags().Changed("file") || cmd.Flags().Changed("body")) {
		return "", fmt.Errorf("Choose one text source: a positional argument, --body, or --file.")
	}
	if cmd.Flags().Changed("body") {
		return c.body, nil
	}
	if cmd.Flags().Changed("file") {
		if c.file == "" {
			return "", fmt.Errorf("--file requires a path or '-' for stdin.")
		}
		return readContent(cmd, c.file)
	}
	arg := ""
	if len(args) > 0 {
		arg = args[0]
	}
	if len(args) == 0 && interactiveInput(cmd) {
		return "", fmt.Errorf("Provide text with --body, --file, or '-' to read stdin. See --help for examples.")
	}
	if positionalFile {
		return readContent(cmd, arg)
	}
	if arg != "" && arg != "-" {
		return strings.TrimSpace(arg), nil
	}
	content, err := readContent(cmd, "-")
	return strings.TrimSpace(content), err
}

func readContent(cmd *cobra.Command, source string) (string, error) {
	var data []byte
	var err error
	if source != "" && source != "-" {
		data, err = os.ReadFile(source)
	} else {
		data, err = io.ReadAll(cmd.InOrStdin())
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func interactiveInput(cmd *cobra.Command) bool {
	file, ok := cmd.InOrStdin().(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}
