package command

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/basecamp/once/internal/docker"
)

type RemoveCommand struct {
	root       *RootCommand
	cmd        *cobra.Command
	removeData bool
}

func NewRemoveCommand(root *RootCommand) *RemoveCommand {
	r := &RemoveCommand{root: root}
	r.cmd = &cobra.Command{
		Use:     "remove <app>",
		Aliases: []string{"rm"},
		Short:   "Remove an application",
		Args:    cobra.ExactArgs(1),
		RunE:    WithNamespace(r.run),
	}
	r.cmd.Flags().BoolVar(&r.removeData, "remove-data", false, "Also remove application data volume")
	return r
}

func (r *RemoveCommand) Command() *cobra.Command {
	return r.cmd
}

// Private

func (r *RemoveCommand) run(ns *docker.Namespace, cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	appName := args[0]

	app := ns.Application(appName)
	if app == nil {
		return fmt.Errorf("application %q not found", appName)
	}

	if err := app.Remove(ctx, r.removeData); err != nil {
		return fmt.Errorf("removing application: %w", err)
	}

	fmt.Printf("Removed %s\n", appName)
	return nil
}
