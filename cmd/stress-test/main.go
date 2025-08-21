package main

import (
	"github.com/JeanGrijp/stress-test/internal/cli"
	"github.com/JeanGrijp/stress-test/internal/commands"
)

func main() {
	root := cli.NewRootCmd()
	// Subcommands
	root.AddCommand(commands.NewVersionCmd())
	root.AddCommand(commands.NewRunCmd())
	root.AddCommand(commands.NewCurlCmd())
	root.AddCommand(commands.NewRampCmd())
	root.AddCommand(commands.NewDocsCmd())

	cli.Execute(root)
}
