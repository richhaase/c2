package terminal

import (
	"os"

	"golang.org/x/term"
)

func IsTerminal(fd int) bool {
	return term.IsTerminal(fd)
}

func IsStdinTTY() bool {
	return IsTerminal(int(os.Stdin.Fd()))
}

func IsStdoutTTY() bool {
	return IsTerminal(int(os.Stdout.Fd()))
}

func IsStderrTTY() bool {
	return IsTerminal(int(os.Stderr.Fd()))
}

func ReadPassword(fd int) (string, error) {
	data, err := term.ReadPassword(fd)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
