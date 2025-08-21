package commands

import (
	"fmt"

	"github.com/JeanGrijp/stress-test/internal/version"
	"github.com/spf13/cobra"
)

// NewVersionCmd returns the `version` subcommand.
func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show CLI version",
		Long: `Print build information for this binary: semantic version, git commit and build date.

Values are embedded at build-time via -ldflags. When not provided, default
placeholders are shown.`,
		Example: `# Show version information
stress-test version`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "version: %s\ncommit: %s\ndate: %s\n", version.Version, version.Commit, version.Date)
		},
	}
	return cmd
}
