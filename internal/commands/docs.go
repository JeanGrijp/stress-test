package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// NewDocsCmd generates CLI docs (markdown or man) from the current command tree.
func NewDocsCmd() *cobra.Command {
	var (
		format string
		outDir string
	)

	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Generate CLI documentation (markdown or man)",
		Long: `Generate documentation files for all commands.

Formats:
  - markdown: one .md file per command
  - man     : one man page per command (section 1)

By default, output is placed under docs/cli (markdown) or docs/man (man).`,
		Example: `# Generate Markdown docs under docs/cli
stress-test docs --format markdown --out-dir ./docs/cli

# Generate man pages under docs/man
stress-test docs --format man --out-dir ./docs/man`,
		RunE: func(cmd *cobra.Command, args []string) error {
			f := format
			switch f {
			case "", "markdown", "md":
				f = "markdown"
			case "man":
				// ok
			default:
				return fmt.Errorf("unsupported --format: %s (use markdown|man)", format)
			}

			// Default outDir if not provided
			if outDir == "" {
				if f == "markdown" {
					outDir = "docs/cli"
				} else {
					outDir = "docs/man"
				}
			}

			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return err
			}

			root := cmd.Root()
			if root == nil {
				return errors.New("root command not found")
			}

			switch f {
			case "markdown":
				// Generate a clean set of markdown files
				return doc.GenMarkdownTree(root, outDir)
			case "man":
				header := &doc.GenManHeader{Title: "STRESS-TEST", Section: "1"}
				if err := doc.GenManTree(root, header, outDir); err != nil {
					return err
				}
				// Optional: create a symlink to stress-test.1 in outDir root for convenience
				src := filepath.Join(outDir, "stress-test.1")
				dst := filepath.Join(outDir, "stress-test")
				_ = os.Remove(dst)
				_ = os.Symlink(src, dst)
				return nil
			default:
				return fmt.Errorf("unexpected format: %s", f)
			}
		},
	}

	cmd.Flags().StringVar(&format, "format", "markdown", "Output format: markdown|man")
	cmd.Flags().StringVar(&outDir, "out-dir", "", "Directory to write files to (default: docs/cli or docs/man)")
	return cmd
}
