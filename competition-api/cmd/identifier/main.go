package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/identifier/cmds"
	workererrors "github.com/aixcyberchallenge/competition-api/competition-api/internal/worker_errors"
)

func runApp(ctx context.Context) int {
	err := cmds.Execute(ctx)
	if err != nil {
		var ee workererrors.ExitError
		if errors.As(err, &ee) {
			return ee.Code
		}

		fmt.Fprintln(os.Stderr, "Error: "+err.Error())
		return 201
	}

	return 0
}

func main() {
	ctx := context.Background()
	os.Exit(runApp(ctx))
}
