package cmd

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"

	"github.com/Guepardo/commit-sync/internal/config"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show mirror configuration and sync status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if cfg.MirrorPath == "" {
			fmt.Println("Mirror: not configured")
			return nil
		}
		fmt.Printf("Mirror: %s\n", cfg.MirrorPath)

		repo, err := git.PlainOpen(cfg.MirrorPath)
		if err != nil {
			fmt.Println("  Status: not initialized or inaccessible")
			return nil
		}

		head, err := repo.Head()
		if err != nil {
			fmt.Println("  Status: empty (no commits yet)")
			return nil
		}

		count := 0
		iter, err := repo.Log(&git.LogOptions{From: head.Hash()})
		if err == nil {
			iter.ForEach(func(c *object.Commit) error {
				count++
				return nil
			})
			iter.Close()
		}

		fmt.Printf("  Branch: %s\n", head.Name().String())
		fmt.Printf("  Commits: %d\n", count)
		fmt.Printf("  Latest: %s\n", head.Hash().String()[:12])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
