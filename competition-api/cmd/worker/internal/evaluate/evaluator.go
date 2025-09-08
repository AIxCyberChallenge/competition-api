package evaluate

import (
	"context"
	"errors"
	"io"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/engine"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/extract"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/workerqueue"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/fetch"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	workererrors "github.com/aixcyberchallenge/competition-api/competition-api/internal/worker_errors"
)

var tracer = otel.Tracer(
	"github.com/aixcyberchallenge/competition-api/competition-api/worker/internal/evaluate",
)

type Evaluator struct {
	fetcher   fetch.Fetcher
	extractor extract.Extractor
	engine    engine.Engine
	queuer    *workerqueue.WorkerQueuer
	tempDir   string
}

func NewEvaluator(
	fetcher fetch.Fetcher,
	extractor extract.Extractor,
	tempDir string,
	engine engine.Engine, //nolint:revive // import-shadowing: no better variable name to use here
	workerQueuer *workerqueue.WorkerQueuer,
) *Evaluator {
	return &Evaluator{
		fetcher:   fetcher,
		extractor: extractor,
		tempDir:   tempDir,
		engine:    engine,
		queuer:    workerQueuer,
	}
}

func (e *Evaluator) Evaluate(
	ctx context.Context,
	fuzzToolingURL, headRepoURL, baseRepoURL, triggerURL, patchURL string, skipPatchTests bool,
	commonEngineParams *engine.Params,
) error {
	ctx, span := tracer.Start(ctx, "Evaluator.Evaluate", trace.WithAttributes(
		attribute.String("fuzzTooling.url", fuzzToolingURL),
		attribute.String("headRepo.url", headRepoURL),
		attribute.String("baseRepo.url", baseRepoURL),
		attribute.String("trigger.url", triggerURL),
		attribute.String("patch.url", patchURL),
	))
	defer span.End()

	status := types.SubmissionStatusPassed
	patchTestsFailed := false

	evalError := make(chan error)

	go func() {
		defer close(evalError)

		err := e.evaluate(
			ctx,
			fuzzToolingURL,
			headRepoURL,
			baseRepoURL,
			triggerURL,
			patchURL,
			skipPatchTests,
			commonEngineParams,
		)
		evalError <- err
	}()

	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			status = types.SubmissionStatusInconclusive
		}
	case err := <-evalError:
		if err != nil {
			var se workererrors.StatusError
			if errors.As(err, &se) {
				status = se.Status
				patchTestsFailed = se.PatchTestsFailed
			} else {
				status = types.SubmissionStatusErrored
			}
		}
		if ctx.Err() == context.DeadlineExceeded {
			status = types.SubmissionStatusInconclusive
		}
	}

	ctx = context.WithoutCancel(ctx)
	err := e.queuer.FinalMessage(ctx, status, patchTestsFailed)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send final message")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "successfully evaluated")
	return nil
}

func (e *Evaluator) evaluate(
	ctx context.Context,
	fuzzToolingURL, headRepoURL, baseRepoURL, triggerURL, patchURL string, skipPatchTests bool,
	commonEngineParams *engine.Params,
) error {
	ctx, span := tracer.Start(ctx, "Evaluator.evaluate")
	defer span.End()

	fuzzToolingDir, err := e.fetchExtractRepo(ctx, fuzzToolingURL)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch fuzzToolingDir")
		return err
	}
	defer os.RemoveAll(fuzzToolingDir)

	headRepoDir, err := e.fetchExtractRepo(ctx, headRepoURL)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch headRepoDir")
		return err
	}
	defer os.RemoveAll(headRepoDir)

	headChallenge := commonEngineParams.
		WithFuzzToolingDir(fuzzToolingDir).
		WithRepo(types.ResultCtxHeadRepoTest, headRepoDir)

	err = e.engine.Check(ctx, &headChallenge)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed check params")
		return err
	}

	var triggerPath string
	if triggerURL != "" {
		var err error
		var trigger *os.File
		trigger, err = e.fetchFile(ctx, triggerURL)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to fetch trigger")
			return err
		}
		defer trigger.Close()
		triggerPath = trigger.Name()

		if patchURL == "" {
			if baseRepoURL != "" {
				if err := e.runBaseTests(ctx, &headChallenge, baseRepoURL, triggerPath); err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, "failed to run base tests")
					return err
				}
			}

			if err := e.runPovTests(ctx, &headChallenge, triggerPath, true); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to run head tests")
				return err
			}
		}
	}

	if patchURL != "" {
		err := e.runPatchTests(
			ctx,
			&headChallenge,
			triggerPath,
			patchURL,
			skipPatchTests,
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to run patch tests")
			return err
		}
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "successfully evaluated")
	return nil
}

// Do not call me for patches. I assume all build failures are errored!
func (e *Evaluator) runPovTests(
	ctx context.Context,
	challenge *engine.Params,
	triggerPath string,
	shouldCrash bool,
) error {
	ctx, span := tracer.Start(ctx, "Evaluator.runPovTests", trace.WithAttributes(
		attribute.String("trigger.path", triggerPath),
		attribute.Bool("shouldCrash", shouldCrash),
	))
	defer span.End()

	err := e.engine.Build(ctx, challenge)
	if err != nil {
		err = workererrors.StatusErrorWrap(types.SubmissionStatusErrored, false, err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to build challenge")
		return err
	}

	err = e.engine.RunPov(ctx, challenge, triggerPath, shouldCrash)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to run pov")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "ran pov tests")
	return nil
}

func (e Evaluator) runBaseTests(
	ctx context.Context,
	headChallenge *engine.Params,
	baseRepoURL, triggerPath string,
) error {
	ctx, span := tracer.Start(ctx, "Evaluator.runBaseTests", trace.WithAttributes(
		attribute.String("baseRepo.url", baseRepoURL),
	))
	defer span.End()

	baseRepoDir, err := e.fetchExtractRepo(ctx, baseRepoURL)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch baseRepo")
		return err
	}
	defer os.RemoveAll(baseRepoDir)

	baseChallenge := headChallenge.WithRepo(types.ResultCtxBaseRepoTest, baseRepoDir)

	err = e.runPovTests(ctx, &baseChallenge, triggerPath, false)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to test baseRepo")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "tested baseRepo")
	return nil
}

func (e *Evaluator) runPatchTests(
	ctx context.Context,
	headChallenge *engine.Params,
	triggerPath, patchURL string,
	skipPatchFunctionalityTests bool,
) error {
	ctx, span := tracer.Start(ctx, "Evaluator.runPatchTests", trace.WithAttributes(
		attribute.String("trigger.path", triggerPath),
		attribute.String("patch.url", patchURL),
	))
	defer span.End()

	patch, err := e.fetchFile(ctx, patchURL)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch patch")
		return err
	}
	defer patch.Close()

	err = e.engine.ApplyPatch(ctx, headChallenge, patch.Name())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to apply patch")
		return err
	}

	err = e.engine.Build(ctx, headChallenge)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to build patch")
		return err
	}

	if triggerPath != "" {
		err = e.engine.RunPov(ctx, headChallenge, triggerPath, false)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to run pov on patch")
			return err
		}
	}

	if !skipPatchFunctionalityTests {
		err = e.engine.RunTests(ctx, headChallenge, true)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to run tests")
			return err
		}
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "ran patch tests")
	return nil
}

func (e *Evaluator) fetchExtractRepo(ctx context.Context, url string) (string, error) {
	ctx, span := tracer.Start(ctx, "Evaluator.fetchExtractRepo", trace.WithAttributes(
		attribute.String("url", url),
	))
	defer span.End()

	repoCompressed, err := e.fetcher.Fetch(ctx, url)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch url")
		return "", err
	}
	defer repoCompressed.Close()

	repoDir, err := os.MkdirTemp(e.tempDir, "repo-*")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get tempdir")
		return "", err
	}

	err = e.extractor.Extract(ctx, repoCompressed, repoDir)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to extract file to tempdir")
		return "", err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "successfully fetched and extracted repo")
	return repoDir, nil
}

func (e *Evaluator) fetchFile(ctx context.Context, url string) (*os.File, error) {
	ctx, span := tracer.Start(ctx, "Evaluator.fetchFile", trace.WithAttributes(
		attribute.String("url", url),
	))
	defer span.End()

	file, err := os.CreateTemp(e.tempDir, "file-*")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to make tempfile")
		return nil, err
	}

	buffer, err := e.fetcher.Fetch(ctx, url)
	if err != nil {
		file.Close()
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch url")
		return nil, err
	}
	defer buffer.Close()

	_, err = io.Copy(file, buffer)
	if err != nil {
		file.Close()
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to copy from body to file")
		return nil, err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "fetched file by url")
	return file, nil
}
