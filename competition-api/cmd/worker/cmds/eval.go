package cmds

import (
	"context"
	"os"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/codes"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/command"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/common"
	workerengine "github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/engine"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/evaluate"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/extract"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/workerqueue"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/fetch"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/identifier"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/upload"
	workererrors "github.com/aixcyberchallenge/competition-api/competition-api/internal/worker_errors"
)

var (
	patchID          string
	patchURL         string
	skipPatchTests   bool
	allowedLanguages identifier.LanguageSlice

	triggerURL  string
	sanitizer   string
	harnessName string
	engine      string

	archiveS3   bool
	baseRepoURL string

	jobID         string
	exportResults bool

	povID          string
	focus          string
	ossFuzzRepoURL string
	projectName    string
	headRepoURL    string
	architecture   string
	baseDir        string
)

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Test all provided inputs (including PoV and Patch, unless patch testing is skipped)",
	Long: `
- Exits with 0 if no errors encountered.
  - true in stdout if all inputs passed all applicable tests
  - false in stdout otherwise
- Exits with a 1 for all other errors.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx, span := tracer.Start(cmd.Context(), "evalCmd")
		defer span.End()

		timeout := (time.Hour * 8)

		logger.Logger.InfoContext(ctx,
			"Starting test job",
			"head-repo",
			headRepoURL,
			"project-name",
			projectName,
			"head-repo-url",
			headRepoURL,
			"fuzz-tooling-url",
			ossFuzzRepoURL,
			"architecture",
			architecture,
			"patch-url",
			patchURL,
			"skip-patch-tests",
			skipPatchTests,
			"allowed-languages",
			allowedLanguages,
			"timeout-duration",
			timeout,
		)

		var entity types.JobType
		var entityID string
		if jobID != "" {
			entity = types.JobTypeJob
			entityID = jobID
		}
		if povID != "" {
			entity = types.JobTypePOV
			entityID = povID
		}
		if patchID != "" {
			entity = types.JobTypePatch
			entityID = patchID
		}

		executor := command.NewShellExecutor()
		queuer, err := common.GetAzureQueueClient()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to make azure queue")
			return err
		}
		workerqueuer := workerqueue.NewWorkerQueue(entityID, entity, queuer)

		httpClient := retryablehttp.NewClient()
		httpClient.RetryMax = 3
		fetcher := fetch.NewHTTPFetcher(httpClient.StandardClient())
		extractor := extract.NewTarGzExtractor(executor)
		azureUploader, err := common.GetAzureBlobClient()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to make azure blob uploader")
			return err
		}

		aixccEngine := workerengine.NewAixccEngine(
			executor,
			upload.NewRetryUploader(azureUploader),
			workerqueuer,
			entity,
		)

		evaluator := evaluate.NewEvaluator(
			fetcher,
			extractor,
			baseDir,
			workerengine.NewBuildRetryEngine(aixccEngine),
			workerqueuer,
		)

		commonEngineParams := workerengine.NewParams(
			sanitizer,
			architecture,
			engine,
			harnessName,
			projectName,
			focus,
			allowedLanguages,
		)
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Hour*8)
		defer cancel()

		err = evaluator.Evaluate(
			timeoutCtx,
			ossFuzzRepoURL,
			headRepoURL,
			baseRepoURL,
			triggerURL,
			patchURL,
			skipPatchTests,
			&commonEngineParams,
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to evaluate")
			return workererrors.ExitErrorWrap(types.ExitErrored, err)
		}

		span.RecordError(err)
		span.SetStatus(codes.Ok, "evaluated successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(evalCmd)

	// Globally required flags
	evalCmd.Flags().StringVar(&headRepoURL, "head-repo-url", "", "Repository tar url (required)")
	evalCmd.Flags().StringVar(&focus, "focus", "", "Main repo (required)")
	evalCmd.Flags().StringVar(&projectName, "project-name", "", "OSS Fuzz project name (required)")
	evalCmd.Flags().StringVar(&ossFuzzRepoURL, "oss-fuzz-url", "", "OSS Fuzz tar url (required)")
	evalCmd.Flags().StringVar(&architecture, "architecture", "", "Architecture (required)")

	for _, requiredFlag := range []string{"head-repo-url", "focus", "project-name", "oss-fuzz-url", "architecture"} {
		err := evalCmd.MarkFlagRequired(requiredFlag)
		if err != nil {
			logger.Logger.Error(
				"error setting flag required",
				"flag",
				requiredFlag,
				"error",
				err,
			)
			os.Exit(types.ExitErrored)
		}
	}

	// Optional flags
	evalCmd.Flags().StringVar(&baseRepoURL, "base-repo-url", "", "Repository tar url")
	evalCmd.Flags().
		StringVar(&baseDir, "base-dir", "", "Base dir to create temporary directories in. Defaults to default temporary directory for the system.")
	evalCmd.Flags().BoolVar(&archiveS3, "archive-s3", false, "Archive files to s3")

	// Job flags
	evalCmd.Flags().
		StringVar(&jobID, "job-id", "", "ID for the Job in the DB.  Only useful in job runner mode.")
	evalCmd.Flags().
		BoolVar(&exportResults, "export-results", false, "Export artifacts to blob storage & log command results on job row")

	// Patch flags
	evalCmd.Flags().StringVar(&patchID, "patch-id", "", "Patch ID")
	evalCmd.Flags().StringVar(&patchURL, "patch-url", "", "Patch url to apply and evaluate.")
	evalCmd.Flags().BoolVar(&skipPatchTests, "skip-patch-tests", false, "Skip patch tests")
	evalCmd.Flags().
		Var(&allowedLanguages, "allowed-languages", "Allowed languages to patch")
	evalCmd.MarkFlagsRequiredTogether("patch-url", "allowed-languages")
	// PoV flags
	evalCmd.Flags().StringVar(&povID, "pov-id", "", "PoV ID")
	evalCmd.Flags().StringVar(&triggerURL, "trigger-url", "", "Trigger URL")
	evalCmd.Flags().StringVar(&sanitizer, "sanitizer", "", "Sanitizer")
	evalCmd.Flags().StringVar(&harnessName, "harness-name", "", "Harness Name")
	evalCmd.Flags().StringVar(&engine, "engine", string(types.FuzzingEngineLibFuzzer), "Engine")
	evalCmd.MarkFlagsRequiredTogether("trigger-url", "sanitizer", "harness-name", "engine")
}
