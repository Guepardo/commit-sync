package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/allyson/commit-sync/internal/config"
	"github.com/allyson/commit-sync/internal/scanner"
	"github.com/allyson/commit-sync/internal/syncer"
)

var dryRun bool

var syncCmd = &cobra.Command{
	Use:   "sync <root-path>",
	Short: "Sync unsynced commits from repos under root-path into the mirror",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if cfg.MirrorPath == "" {
			return fmt.Errorf("mirror path not configured; use 'commit-sync set-mirror <path>' first")
		}

		results, err := scanner.Scan(root, cfg.MirrorPath)
		if err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		if len(results) == 0 {
			fmt.Println("No repositories without remotes found.")
			return nil
		}

		s := syncer.New(cfg.MirrorPath)
		s.SetDryRun(dryRun)

		n, err := s.Sync(results)
		if err != nil {
			return fmt.Errorf("sync: %w", err)
		}

		if dryRun {
			fmt.Printf("Would sync %d commit(s) from %d repo(s).\n", n, len(results))
		} else {
			fmt.Printf("Synced %d commit(s) from %d repo(s).\n", n, len(results))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be synced without making changes")
}
