package cmd

import (
	"github.com/spf13/cobra"
)

// repoCmd represents the repo command
var repoCmd = &cobra.Command{
	Use:     "repo",
	Aliases: []string{"project"},
	Short:   "Perform project level operations on GitLab",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	RootCmd.AddCommand(repoCmd)
}
