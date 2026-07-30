package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/allyson/commit-sync/internal/config"
)

var setMirrorCmd = &cobra.Command{
	Use:   "set-mirror <path>",
	Short: "Configure the mirror repository path",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		absPath, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("resolve path: %w", err)
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		cfg.MirrorPath = absPath

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("Mirror path set to: %s\n", absPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(setMirrorCmd)
}
