package cmds

import (
	"context"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "worker",
	Short: "Worker command to do all tasks for k8s jobs",
}

func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}
