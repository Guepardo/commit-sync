package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "commit-sync",
	Version: version,
	Short:   "Sync commits from local git repos without remotes into a mirror repository",
	Long: `commit-sync discovers git repositories without remotes, collects their commits,
and replicates them into a single mirror repository in chronological order.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
