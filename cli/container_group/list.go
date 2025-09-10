/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container_group

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

type ComposeProject struct {
	Name string
	Dir  string
}

func (cp *ComposeProject) GetVersion() string {
	file, err := os.Open(filepath.Join(cp.Dir, "version"))
	if err != nil {
		return "latest"
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	return scanner.Text()
}

// listCmd represents the list command
func NewListCommand(cliContext cli.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List containers",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)
			ctx := context.Background()
			cli, err := container.NewContainerClient(context.TODO())
			if err != nil {
				return err
			}
			containers, err := cli.List(ctx, container.FilterOptions{
				Labels: []string{"com.docker.compose.project"},
			})
			if err != nil {
				return err
			}
			stdout := cmd.OutOrStdout()
			projects := make(map[string]ComposeProject)
			for _, item := range containers {
				if project, ok := item.Container.Labels["com.docker.compose.project"]; ok {
					projects[project] = ComposeProject{
						Name: project,
						Dir:  item.Container.Labels["com.docker.compose.project.working_dir"],
					}
				}
			}
			keys := make([]string, 0, len(projects))
			for project := range projects {
				keys = append(keys, project)
			}
			sort.Strings(keys)

			for _, key := range keys {
				project := projects[key]
				fmt.Fprintf(stdout, "%s\t%s\n", project.Name, project.GetVersion())
			}
			return nil
		},
	}
}
