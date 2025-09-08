package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v2"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/command"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/workerqueue"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/common"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/identifier"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/upload"
	workererrors "github.com/aixcyberchallenge/competition-api/competition-api/internal/worker_errors"
)

// Ensure AixccEngine implementes Engine interface
var _ Engine = (*AixccEngine)(nil)

type AixccEngine struct {
	executor         command.Executor
	artifactUploader upload.Uploader
	workerqueuer     *workerqueue.WorkerQueuer
	entity           types.JobType
}

func NewAixccEngine(
	executor command.Executor,
	artifactUploader upload.Uploader,
	workerqueuer *workerqueue.WorkerQueuer,
	entity types.JobType,

) *AixccEngine {
	return &AixccEngine{
		executor:         executor,
		artifactUploader: artifactUploader,
		workerqueuer:     workerqueuer,
		entity:           entity,
	}
}

func (c *AixccEngine) Check(ctx context.Context, data *Params) error {
	ctx, span := tracer.Start(ctx, "AixccChallenge.Check", trace.WithAttributes(
		attribute.String("data.repoDir", data.repoDir),
		attribute.String("data.fuzzToolingDir", data.fuzzToolingDir),
		attribute.String("data.focus", data.focus),
		attribute.String("data.projectName", data.projectName),
	))
	defer span.End()

	if data.repoDir == "" || data.fuzzToolingDir == "" || data.focus == "" ||
		data.projectName == "" {
		err := errors.New("missing source path information")
		span.RecordError(err)
		span.SetStatus(codes.Error, "missing source path information")
		return err
	}

	fuzzToolingDir, err := c.getFuzzToolingDir(ctx, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get internal fuzztooling dir")
		return err
	}

	if data.architecture != string(types.ArchitectureX8664) {
		err = fmt.Errorf("invalid architecture: %s", data.architecture)
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid architecture")
		return workererrors.StatusErrorWrap(types.SubmissionStatusFailed, false, err)
	}

	if data.engine != string(types.FuzzingEngineLibFuzzer) {
		err = fmt.Errorf("invalid engine: %s", data.engine)
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid engine")
		return workererrors.StatusErrorWrap(types.SubmissionStatusFailed, false, err)
	}

	type projectYAML struct {
		Language   string   `yaml:"language"`
		Sanitizers []string `yaml:"sanitizers"`
	}
	projectYAMLPath := filepath.Join(fuzzToolingDir, "projects", data.projectName, "project.yaml")
	projectYamlData, err := os.ReadFile(projectYAMLPath)
	if err != nil {
		span.RecordError(err, trace.WithAttributes(attribute.String("path", projectYAMLPath)))
		span.SetStatus(codes.Error, "failed to read project.yaml")
		return err
	}
	p := projectYAML{}
	err = yaml.Unmarshal(projectYamlData, &p)
	if err != nil {
		span.RecordError(
			err,
			trace.WithAttributes(attribute.String("data", string(projectYamlData))),
		)
		span.SetStatus(codes.Error, "failed to unmarshal project.yaml")
		return err
	}

	if p.Language == "jvm" {
		settings, err := os.Open("./settings.xml")
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "faile to open settings")
			return err
		}
		defer settings.Close()

		projectLocalSettings, err := os.Create(
			filepath.Join(fuzzToolingDir, "projects", data.projectName, "settings.xml"),
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to open project local settings")
			return err
		}
		defer projectLocalSettings.Close()

		_, err = io.Copy(projectLocalSettings, settings)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to override helper py")
			return err
		}

		dockerfile, err := os.OpenFile(
			filepath.Join(fuzzToolingDir, "projects", data.projectName, "Dockerfile"),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND,
			0666,
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to open dockerfile")
			return err
		}
		defer dockerfile.Close()

		fmt.Fprint(dockerfile, "\nCOPY settings.xml /root/.m2/settings.xml\n")
	}

	if data.sanitizer != "" && !slices.Contains(p.Sanitizers, data.sanitizer) {
		err = fmt.Errorf("invalid sanitizer: %s", data.sanitizer)
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid sanitizer")
		return workererrors.StatusErrorWrap(types.SubmissionStatusFailed, false, err)
	}

	e, err := types.ParseChallengeYAML(ctx, filepath.Join(data.repoDir, data.focus))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to decode challenge yaml")
		return err
	}

	if data.harness != "" && !slices.Contains(e.HarnessesList, data.harness) {
		err = fmt.Errorf("invalid harness: %s", data.harness)
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid harness")
		return workererrors.StatusErrorWrap(types.SubmissionStatusFailed, false, err)
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "checked args")
	return nil
}

func (c *AixccEngine) Build(
	ctx context.Context,
	data *Params,
) error {
	ctx, span := tracer.Start(ctx, "AixccChallenge.Build", trace.WithAttributes(
		attribute.String("data.architecture", data.architecture),
		attribute.String("data.projectName", data.projectName),
		attribute.String("data.repoDir", data.repoDir),
		attribute.String("data.fuzzToolingDir", data.fuzzToolingDir),
		attribute.String("data.focus", data.focus),
	))
	defer span.End()

	if data.architecture == "" ||
		data.projectName == "" ||
		data.repoDir == "" ||
		data.fuzzToolingDir == "" ||
		data.focus == "" {
		err := errors.New("missing call params")
		span.RecordError(err)
		span.SetStatus(codes.Error, "missing call params")
		return err
	}

	fuzzToolingDir, err := c.getFuzzToolingDir(ctx, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get fuzztooling dir")
		return err
	}

	args := make([]string, 0, 10)
	args = append(args, "-e")

	if data.sanitizer != "" {
		// TODO: warn about missing sanitizer
		args = append(args, "-s", data.sanitizer)
	}

	args = append(args,
		"-a", data.architecture,
		"-p", data.projectName,
		"-r", filepath.Join(data.repoDir, data.focus),
		"-o", fuzzToolingDir,
	)

	cmd := command.New("./build_cr.sh", args...)
	result, err := c.executor.Execute(ctx, cmd)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to execute build")
		return err
	}

	if result.ExitCode == 202 {
		if strings.Contains(string(result.Stderr), "Unable to fetch some archives") {
			span.RecordError(ErrAptUnreachable)
			span.SetStatus(codes.Error, "failed to fetch some archives")
			return ErrAptUnreachable
		} else if strings.Contains(string(result.Stdout), "Could not transfer") {
			span.RecordError(ErrMavenUnreachable)
			span.SetStatus(codes.Error, "failed to reach maven")
			return ErrMavenUnreachable
		}

		span.RecordError(ErrBuildingFailed)
		span.SetStatus(codes.Error, "got bad exit code from build")
		return workererrors.StatusErrorWrap(types.SubmissionStatusFailed, false, ErrBuildingFailed)
	} else if result.ExitCode != 0 {
		err = errors.New("problem executing build")
		span.RecordError(err)
		span.SetStatus(codes.Error, "problem executing build")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "successfully built cr")
	return nil
}

func (c *AixccEngine) RunPov(
	ctx context.Context,
	data *Params,
	triggerPath string,
	crashExpected bool,
) error {
	ctx, span := tracer.Start(ctx, "AixccChallenge.RunPov", trace.WithAttributes(
		attribute.String("data.architecture", data.architecture),
		attribute.String("data.projectName", data.projectName),
		attribute.String("data.fuzzToolingDir", data.fuzzToolingDir),
		attribute.String("data.harness", data.harness),
		attribute.String("data.engine", data.engine),
		attribute.String("data.sanitizer", data.sanitizer),
		attribute.String("data.resultContext", string(data.resultContext)),
		attribute.String("triggerPath", triggerPath),
		attribute.Bool("crashExpected", crashExpected),
	))
	defer span.End()

	if data.architecture == "" ||
		data.projectName == "" ||
		data.fuzzToolingDir == "" ||
		data.harness == "" ||
		data.engine == "" ||
		data.sanitizer == "" ||
		triggerPath == "" ||
		data.resultContext == "" {
		err := errors.New("missing call params")
		span.RecordError(err)
		span.SetStatus(codes.Error, "missing call params")
		return err
	}

	fuzzToolingDir, err := c.getFuzzToolingDir(ctx, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get fuzztoolingdir")
		return err
	}

	args := make([]string, 0, 20)
	if !crashExpected {
		args = append(args, "-x")
	}

	args = append(
		args,
		"-a", data.architecture,
		"-p", data.projectName,
		"-o", fuzzToolingDir,
		"-f", data.harness,
		"-e", data.engine,
		"-s", data.sanitizer,
		"-b", triggerPath,
		"-n",
		// Seconds
		"-t", "1800",
	)

	loopCount := 1
	if !crashExpected {
		loopCount = 3
	}

	var result *command.Result
	cmd := command.New("./run_pov.sh", args...)
	for range loopCount {
		span.SetAttributes(attribute.Int("max_loop_count", loopCount))

		result, err = c.executor.Execute(ctx, cmd)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to execute command")
			return err
		}

		if !crashExpected && result.ExitCode == 202 {
			span.AddEvent("crashed_with_patch")
			break
		}
	}

	stdoutHash, err := upload.Hashed(
		ctx,
		c.artifactUploader,
		bytes.NewReader(result.Stdout),
		int64(len(result.Stdout)),
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to upload stdout")
		return err
	}
	stderrHash, err := upload.Hashed(
		ctx,
		c.artifactUploader,
		bytes.NewReader(result.Stderr),
		int64(len(result.Stderr)),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to upload stderr")
		return err
	}

	err = c.workerqueuer.CommandResult(ctx, &types.JobResult{
		Cmd:        result.Cmd,
		StdoutBlob: types.Blob{ObjectName: stdoutHash},
		StderrBlob: types.Blob{ObjectName: stderrHash},
		ExitCode:   &result.ExitCode,
		Context:    data.resultContext,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to queue command result")
		return err
	}

	exists := true
	stat, err := os.Stat("/tmp/fuzz.out")
	if err != nil {
		if !os.IsNotExist(err) {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to stat file")
			return err
		}

		exists = false
	}

	if exists {
		fuzzOutFile, err := os.Open("/tmp/fuzz.out")
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to open fuzz out")
			return err
		}
		defer fuzzOutFile.Close()

		fuzzOutHash, err := upload.Hashed(ctx, c.artifactUploader, fuzzOutFile, stat.Size())
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to upload fuzz out")
			return err
		}

		archivedFile := types.FileFuzzOutHead
		if data.resultContext == types.ResultCtxBaseRepoTest {
			archivedFile = types.FileFuzzOutBase
		}

		err = c.workerqueuer.Artifact(ctx, types.JobArtifact{
			Blob:         types.Blob{ObjectName: fuzzOutHash},
			Context:      data.resultContext,
			Filename:     fuzzOutFile.Name(),
			ArchivedFile: archivedFile,
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to queue artifact")
			return err
		}
	}

	// the linter thinks this is better...
	switch result.ExitCode {
	case 0:
	case 202:
		span.RecordError(nil)
		span.SetStatus(codes.Error, "script error")
		return workererrors.StatusErrorWrap(types.SubmissionStatusFailed, false, nil)
	default:
		err = fmt.Errorf("unexpected exit code: %d", result.ExitCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, "unexpected exit code")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "ran pov")
	return nil
}

func removePatchPrefix(ctx context.Context, input string) string {
	_, span := tracer.Start(
		ctx,
		"removePatchPrefix",
		trace.WithAttributes(attribute.String("input", input)),
	)
	defer span.End()

	for _, pfx := range []string{"a/", "b/"} {
		if strings.HasPrefix(input, pfx) {
			span.SetAttributes(
				attribute.Bool("found_prefix", true),
				attribute.String("prefix", pfx),
			)
			return strings.TrimPrefix(input, pfx)
		}
	}

	span.SetAttributes(attribute.Bool("found_prefix", false))
	return input
}

func (c *AixccEngine) ApplyPatch(
	ctx context.Context,
	data *Params,
	patchPath string,
) error {
	ctx, span := tracer.Start(ctx, "AixccChallenge.ApplyPatch", trace.WithAttributes(
		attribute.String("patchPath", patchPath),
		attribute.String("data.repoDir", data.repoDir),
		attribute.String("data.focus", data.focus),
		attribute.StringSlice(
			"data.allowedLanguages",
			common.SliceToStringSlice(data.allowedLanguages),
		),
	))
	defer span.End()

	if len(data.allowedLanguages) == 0 || data.repoDir == "" ||
		data.focus == "" {
		err := errors.New("missing call params")
		span.RecordError(err)
		span.SetStatus(codes.Error, "missing call params")
		return err
	}

	files, err := parsePatch(patchPath)
	if err != nil {
		span.RecordError(nil)
		span.SetStatus(codes.Error, "invalid patch file")
		return workererrors.StatusErrorWrap(types.SubmissionStatusFailed, false, nil)
	}

	allowed := true
	for _, file := range files {
		if file.IsNew {
			continue
		}

		fileAllowed, err := c.checkFile(
			ctx,
			removePatchPrefix(ctx, file.OldName),
			data,
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to check file")
			return err
		}

		allowed = allowed && fileAllowed
	}

	if !allowed {
		err = workererrors.StatusErrorWrap(
			types.SubmissionStatusFailed,
			false,
			errors.New("file modifies forbidden language"),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, "file modifies forbidden language")
		return err
	}

	cmd := command.New(
		"git",
		"-C",
		filepath.Join(data.repoDir, data.focus),
		"apply",
		patchPath,
	)
	result, err := c.executor.Execute(ctx, cmd)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to execute patch command")
		return err
	}

	allowed = true
	for _, file := range files {
		if file.IsDelete {
			continue
		}

		fileAllowed, err := c.checkFile(
			ctx,
			removePatchPrefix(ctx, file.NewName),
			data,
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to check file")
			return err
		}

		allowed = allowed && fileAllowed
	}

	if !allowed {
		err = workererrors.StatusErrorWrap(
			types.SubmissionStatusFailed,
			false,
			errors.New("file modifies forbidden language"),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, "file modifies forbidden language")
		return err
	}

	// I could not find exit code docs for git-apply. For now handle all non zero codes as the patch submitters fault.
	// If we can find docs for this we can handle things fine grained.
	if result.ExitCode != 0 {
		span.RecordError(nil)
		span.SetStatus(codes.Error, "invalid patch exit code")
		return workererrors.StatusErrorWrap(types.SubmissionStatusFailed, false, nil)
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "applied patch")
	return nil
}

func (c *AixccEngine) RunTests(
	ctx context.Context,
	data *Params,
	successExpected bool,
) error {
	ctx, span := tracer.Start(ctx, "AixccChallenge.RunTests", trace.WithAttributes(
		attribute.Bool("successExpected", successExpected),
		attribute.String("data.projectName", data.projectName),
		attribute.String("data.repoDir", data.repoDir),
		attribute.String("data.focus", data.focus),
	))
	defer span.End()

	if data.projectName == "" ||
		data.repoDir == "" ||
		data.focus == "" {
		err := errors.New("missing call params")
		span.RecordError(err)
		span.SetStatus(codes.Error, "missing call params")
		return err
	}

	args := make([]string, 0, 6)
	if !successExpected {
		args = append(args, "-x")
	}

	args = append(
		args,
		"-p", data.projectName,
		"-r", filepath.Join(data.repoDir, data.focus),
	)

	cmd := command.New("./run_tests.sh", args...)
	result, err := c.executor.Execute(ctx, cmd)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to execute command")
		return err
	}

	if result.ExitCode == 202 {
		span.RecordError(nil)
		span.SetStatus(codes.Error, "tests did not match expected state")
		return workererrors.StatusErrorWrap(types.SubmissionStatusFailed, true, nil)
	} else if result.ExitCode != 0 {
		err = fmt.Errorf("unexpected exit code: %d", result.ExitCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, "unexpected exit code")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "successfully ran tests")
	return nil
}

func parsePatch(patchPath string) ([]*gitdiff.File, error) {
	patchFile, err := os.Open(patchPath)
	if err != nil {
		return nil, fmt.Errorf("error opening patch file: %w", err)
	}
	defer patchFile.Close()

	files, _, err := gitdiff.Parse(patchFile)
	if err != nil {
		return nil, fmt.Errorf("error parsing patch file: %w", err)
	}

	for _, file := range files {
		file.NewName = strings.TrimSpace(file.NewName)
		file.OldName = strings.TrimSpace(file.OldName)
	}

	return files, nil
}

// true if file is allowed
func (*AixccEngine) checkFile(ctx context.Context, path string, data *Params) (bool, error) {
	_, span := tracer.Start(ctx, "AixccEngine.checkFile", trace.WithAttributes(
		attribute.String("path", path),
		attribute.String("repoDir", data.repoDir),
		attribute.String("focus", data.focus),
	))
	defer span.End()

	content, err := os.ReadFile(filepath.Join(data.repoDir, data.focus, path))
	if err != nil {
		if os.IsNotExist(err) {
			span.RecordError(err)
			span.SetStatus(codes.Error, "file does not exist")
			return false, workererrors.StatusErrorWrap(types.SubmissionStatusFailed, false, err)
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read file")
		return false, err
	}

	language := identifier.GetLanguage(path, content)
	span.SetAttributes(attribute.String("detected_language", language.String()))

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "checked language")
	return slices.Contains(data.allowedLanguages, language), nil
}

func (c *AixccEngine) getFuzzToolingDir(ctx context.Context, data *Params) (string, error) {
	ctx, span := tracer.Start(ctx, "getFuzzToolingDir")
	l := logger.Logger.With("fuzzToolingDir", data.fuzzToolingDir)
	defer span.End()

	l.DebugContext(ctx, "reading dir")
	entries, err := os.ReadDir(data.fuzzToolingDir)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read fuzztooling dir")
		return "", err
	}
	if len(entries) == 0 {
		err = errors.New("invalid format for fuzz tooling tar")
		span.RecordError(err)
		span.SetStatus(codes.Error, "fuzz tooling dir was empty")
		return "", err
	}

	l.DebugContext(ctx, "choosing first entry", "entryName", entries[0].Name())
	fuzzToolingDir := filepath.Join(data.fuzzToolingDir, entries[0].Name())

	if c.entity == types.JobTypeJob {
		override, err := os.Open("./helper.py")
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to open override helper py ")
			return "", err
		}
		defer override.Close()

		helper, err := os.Create(filepath.Join(fuzzToolingDir, "infra", "helper.py"))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to open helper py for override")
			return "", err
		}
		defer helper.Close()

		_, err = io.Copy(helper, override)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to override helper py")
			return "", err
		}
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "calculated fuzz dir")
	return fuzzToolingDir, nil
}
