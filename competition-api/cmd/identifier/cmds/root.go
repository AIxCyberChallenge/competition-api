package cmds

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/identifier"
	workererrors "github.com/aixcyberchallenge/competition-api/competition-api/internal/worker_errors"
)

var (
	language identifier.Language
	path     string
)

var rootCmd = &cobra.Command{
	Use:           "identifier",
	Short:         "Identifies file type",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		found := identifier.GetLanguage(path, content)
		fmt.Println(found == language)

		code := 1
		if found == language {
			code = 0
		}

		return workererrors.ExitErrorWrap(code, nil)
	},
}

func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.Flags().Var(&language, "language", `"java" or "c"`)
	rootCmd.Flags().StringVar(&path, "path", "", "Path to file to check")
	for _, flag := range []string{"language", "path"} {
		err := rootCmd.MarkFlagRequired(flag)
		if err != nil {
			panic("Internal error contact a contributor [path-flag-required]")
		}
	}
}
