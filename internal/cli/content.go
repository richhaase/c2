package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/terminal"
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
	return readBody(cmd, arg)
}

func interactiveInput(cmd *cobra.Command) bool {
	file, ok := cmd.InOrStdin().(*os.File)
	return ok && terminal.IsTerminal(int(file.Fd()))
}
