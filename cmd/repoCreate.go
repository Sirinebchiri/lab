package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// repoCreateCmd represents the create command
var repoCreateCmd = &cobra.Command{
	Use:   "create [path]",
	Short: "Create a new project on GitLab",
	Long:  ``,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var (
			name, _ = cmd.Flags().GetString("name")
			desc, _ = cmd.Flags().GetString("description")
			path    string
		)
		if len(args) > 0 {
			path = args[0]
		}
		// TODO: allow these to be empty if we are in a initialized git repo
		if path == "" && name == "" {
			log.Fatal("path or name must be set")
		}

		opts := gitlab.CreateProjectOptions{
			Path:        gitlab.String(path),
			Name:        gitlab.String(name),
			Description: gitlab.String(desc),
		}
		p, err := lab.RepoCreate(&opts)
		if err != nil {
			log.Fatal(err)
		}
		// TODO: only do this in an empty git repo (one without
		// remotes). This way the command can be run safely in existing
		// repos or outside of a git repo entirely
		err = git.RemoteAdd("origin", p.SSHURLToRepo, ".")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(p.HTTPURLToRepo)
	},
}

func init() {
	repoCreateCmd.Flags().StringP("name", "n", "", "name to use for the new project")
	repoCreateCmd.Flags().StringP("description", "d", "", "description to use for the new project")
	repoCmd.AddCommand(repoCreateCmd)
}
