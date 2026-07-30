package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/allyson/commit-sync/internal/config"
	"github.com/allyson/commit-sync/internal/scanner"
)

var scanCmd = &cobra.Command{
	Use:   "scan <root-path>",
	Short: "Scan a directory for git repos without remotes",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		results, err := scanner.Scan(root, cfg.MirrorPath)
		if err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		if len(results) == 0 {
			fmt.Println("No repositories without remotes found.")
			return nil
		}

		fmt.Printf("Found %d repo(s) without remotes:\n\n", len(results))
		for _, r := range results {
			fmt.Printf("  %s\n", r.Path)
			fmt.Printf("    Branch: %s\n", r.DefaultBranch)
			fmt.Printf("    Commits: %d\n", r.CommitCount)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
