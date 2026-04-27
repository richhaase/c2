package main

import (
	"fmt"
	"os"

	"github.com/richhaase/c2/internal/cli"
)

func main() {
	if err := cli.NewRootCommand(buildVersionString(), cli.DefaultDependencies()).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
