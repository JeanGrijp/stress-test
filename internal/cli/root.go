package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewRootCmd creates the CLI root command.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stress-test",
		Short: "CLI to run load/stress tests",
		Long: `stress-test is a simple, fast CLI for HTTP load testing.

Use the subcommands to run different kinds of tests:
	- run   : Fire a fixed number of requests with a given concurrency
	- ramp  : Execute multiple phases ramping concurrency (by requests, duration, or target RPS)
	- curl  : Send a single HTTP request using a small subset of curl flags
	- version: Print build information (version, commit, date)

Global flags:
	-v, --verbose   Verbose mode for additional logs to stderr

Tip: append --help to any subcommand to see its specific flags and examples.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Example: `# Basic: run 100 requests with concurrency 10
stress-test run --url https://example.com --requests 100 --concurrency 10

# Ramp up concurrency in 3 phases of 10 seconds each, targeting 50rps then +10 per step
stress-test ramp --url https://example.com --steps 3 --start-concurrency 5 \
	--step-concurrency 5 --per-step-duration 10s --rps 50 --step-rps 10

# Single request like curl and print response headers
stress-test curl -i https://httpbin.org/get

# Show version/build info
stress-test version`,
	}

	// Global flags
	cmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose mode")

	return cmd
}

// Execute runs the root command and handles errors consistently.
func Execute(root *cobra.Command) {
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
