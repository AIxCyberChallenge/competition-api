package challenges

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/uuid"
	cp "github.com/otiai10/copy"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/jobs"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/archive"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/config"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/upload"
)

var tracer = otel.Tracer(
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/challenges",
)

type Client struct {
	archiver     upload.Uploader
	workingStore upload.Uploader
	db           *gorm.DB
	jobClient    *jobs.KubernetesClient
	tempDir      string
}

func Create(
	db *gorm.DB,
	tempDir string,
	jobClient *jobs.KubernetesClient,
	archiver upload.Uploader,
	workingStore upload.Uploader,
) *Client {
	return &Client{
		db:           db,
		tempDir:      tempDir,
		jobClient:    jobClient,
		archiver:     archiver,
		workingStore: workingStore,
	}
}

func command(
	ctx context.Context,
	workDir string,
	ignoreReturnCodes []int,
	name string,
	args ...string,
) (string, string, error) {
	ctx, span := tracer.Start(ctx, "command")
	defer span.End()

	span.SetAttributes(
		attribute.String("command", name),
		attribute.StringSlice("args", args),
	)

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	span.SetAttributes(
		attribute.String("stdout", stdout.String()),
		attribute.String("stderr", stderr.String()),
	)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			span.SetAttributes(attribute.Int("returnCode", exitErr.ExitCode()))

			if slices.Contains(ignoreReturnCodes, exitErr.ExitCode()) {
				span.AddEvent("Ignoring return code error")
				err = nil
				span.SetStatus(codes.Ok, "")
				span.RecordError(nil)
			} else {
				span.SetStatus(codes.Error, "bad return code from command")
				span.RecordError(err)
			}
		} else {
			span.SetStatus(codes.Error, "error running command")
			span.RecordError(err)
		}
	} else {
		span.SetStatus(codes.Ok, "")
		span.RecordError(nil)
	}

	return stdout.String(), stderr.String(), err
}

// Generate a challenge task
// Set up workspace
func setupWorkspace(
	ctx context.Context,
	baseDir string,
	internalWorkDirs ...string,
) (string, error) {
	_, span := tracer.Start(ctx, "setupWorkspace")
	defer span.End()

	span.SetAttributes(attribute.String("baseDir", baseDir))

	tmpdir, err := os.MkdirTemp(baseDir, "challengetask")
	if err != nil {
		span.SetStatus(codes.Error, "error creating temp dir")
		span.RecordError(err)
		return "", err
	}

	span.AddEvent("creating work dirs")
	for _, dir := range internalWorkDirs {
		span.AddEvent("creating dir", trace.WithAttributes(attribute.String("dir", dir)))
		err = os.MkdirAll(filepath.Join(tmpdir, dir), 0700)
		if err != nil {
			span.SetStatus(codes.Error, "error creating temp dir")
			span.RecordError(err)
			return "", err
		}
	}

	span.SetStatus(codes.Ok, "")
	span.RecordError(nil)
	return tmpdir, nil
}

func getAuthenticatedURL(
	ctx context.Context,
	input string,
	auth *githttp.BasicAuth,
) (string, error) {
	_, span := tracer.Start(ctx, "getAuthenticatedURL")
	defer span.End()

	span.SetAttributes(attribute.String("input", input))

	span.AddEvent("parsing url")
	parsedURL, err := url.Parse(input)
	if err != nil {
		span.SetStatus(codes.Error, "error parsing input")
		span.RecordError(err)
		return "", err
	}

	if auth != nil {
		parsedURL.User = url.UserPassword(auth.Username, auth.Password)
	} else {
		parsedURL.User = nil
	}

	span.SetStatus(codes.Ok, "")
	span.RecordError(nil)
	return parsedURL.String(), nil
}

func setRepoURL(
	ctx context.Context,
	repoPath string,
	oldURL string,
	auth *githttp.BasicAuth,
) error {
	_, span := tracer.Start(ctx, "setRepoURL")
	defer span.End()

	span.SetAttributes(
		attribute.String("repoPath", repoPath),
		attribute.String("oldURL", oldURL),
	)

	oldURL = replaceSSH(oldURL)

	newURL, err := getAuthenticatedURL(ctx, oldURL, auth)
	if err != nil {
		span.SetStatus(codes.Error, "failed to build auth URL")
		span.RecordError(err)
		return err
	}

	span.AddEvent("setting up new URL for origin")
	_, _, err = command(
		ctx,
		"",
		[]int{},
		"git",
		"-C",
		repoPath,
		"remote",
		"set-url",
		"origin",
		newURL,
	)
	if err != nil {
		span.SetStatus(codes.Error, "failed to set origin")
		span.RecordError(err)
		return err
	}

	span.SetStatus(codes.Ok, "")
	span.RecordError(nil)
	return nil
}

// parse repo URL
func extractRepoName(ctx context.Context, input string) (string, error) {
	_, span := tracer.Start(ctx, "extractRepoName")
	defer span.End()

	span.SetAttributes(attribute.String("extractRepoName.input", input))

	span.AddEvent("parsing input as url")
	parsedURL, err := url.Parse(input)
	if err != nil {
		span.SetStatus(codes.Error, "error parsing input")
		span.RecordError(err)
		return "", err
	}

	span.AddEvent("Extracting repo name from URL")
	path := strings.TrimSuffix(parsedURL.Path, "/") // Remove trailing slash if any
	lastSegment := path[strings.LastIndex(path, "/")+1:]
	splitString := strings.Split(lastSegment, ".")
	lastSegment = splitString[0]

	span.SetAttributes(attribute.String("repo.name", lastSegment))

	span.SetStatus(codes.Ok, "")
	span.RecordError(nil)
	return lastSegment, nil
}

// Get repo:
func cloneRepo(ctx context.Context, url string, path string, auth githttp.BasicAuth) error {
	_, span := tracer.Start(ctx, "cloneRepo")
	defer span.End()

	span.SetAttributes(
		attribute.String("repo.url", url),
		attribute.String("repo.path", path),
	)

	_, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:  url,
		Auth: &auth,
	})
	if err != nil {
		span.SetStatus(codes.Error, "error cloning repo")
		span.RecordError(err)
		return err
	}

	span.AddEvent("setting up LFS-capable authenticated URL for origin")
	err = setRepoURL(ctx, path, url, &auth)
	if err != nil {
		span.SetStatus(codes.Error, "failed to set origin to authenticated URL")
		span.RecordError(err)
		return err
	}

	span.SetStatus(codes.Ok, "")
	span.RecordError(nil)
	return nil
}

func downloadRepo(
	ctx context.Context,
	repoURL string,
	downloadPath string,
	auth githttp.BasicAuth,
) (string, error) {
	ctx, span := tracer.Start(ctx, "downloadRepo")
	defer span.End()

	span.SetAttributes(
		attribute.String("repo.url", repoURL),
		attribute.String("repo.downloadPath", downloadPath),
	)

	repoURL = replaceSSH(repoURL)

	repoName, err := extractRepoName(ctx, repoURL)
	if err != nil {
		span.SetStatus(codes.Error, "error extracting repo name")
		span.RecordError(err)
		return "", err
	}

	repoDir := path.Join(downloadPath, repoName)
	span.SetAttributes(attribute.String("repo.dir", repoDir))
	span.AddEvent("creating repo dir")
	err = os.Mkdir(repoDir, 0700)
	if err != nil {
		span.SetStatus(codes.Error, "failed to create repo dir")
		span.RecordError(err)
		return "", err
	}

	err = cloneRepo(ctx, repoURL, repoDir, auth)
	if err != nil {
		span.SetStatus(codes.Error, "error cloning repo")
		span.RecordError(err)
		return "", err
	}

	span.SetStatus(codes.Ok, "")
	span.RecordError(nil)
	return repoDir, nil
}

// paths to strip are relative to repo root
func stripRepo(ctx context.Context, pathsToStrip []string, repoPath string) error {
	_, span := tracer.Start(ctx, "stripRepo")
	defer span.End()

	span.SetAttributes(attribute.String("repo.path", repoPath))

	for _, target := range pathsToStrip {
		span.AddEvent("strip repo path", trace.WithAttributes(attribute.String("target", target)))
		err := os.RemoveAll(path.Join(repoPath, target))
		if err != nil {
			span.SetStatus(codes.Error, "error cloning repo")
			span.RecordError(err)
			return err
		}
	}

	span.SetStatus(codes.Ok, "")
	span.RecordError(nil)
	return nil
}

func packageRepo(ctx context.Context, srcDir string, tarsDir string) (string, error) {
	ctx, span := tracer.Start(ctx, "packageRepo")
	defer span.End()

	span.SetAttributes(
		attribute.String("srcDir", srcDir),
		attribute.String("tarsDir", tarsDir),
	)

	span.AddEvent("check source directory exists")
	// TODO handle other possible errors here
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		newErr := fmt.Errorf("source directory does not exist: %s", srcDir)
		span.SetStatus(codes.Error, "source directory does not exist")
		span.RecordError(newErr)
		return "", newErr
	}

	span.AddEvent("make tarball name")
	tarballName := uuid.New().String() + ".tar.gz"
	tarballPath := path.Join(tarsDir, tarballName)

	span.SetAttributes(attribute.String("tarball.path", tarballPath))

	span.AddEvent("create tarball")
	// https://www.gnu.org/software/tar/manual/html_node/Reproducibility.html
	// https://web.archive.org/web/20250427022906/https://www.gnu.org/software/tar/manual/html_node/Reproducibility.html
	_, _, err := command(
		ctx,
		"",
		[]int{},
		"tar",
		"-I",
		"gzip --no-name --best",
		"--sort=name",
		"--format=posix",
		"--pax-option=exthdr.name=%d/PaxHeaders/%f",
		"--pax-option=delete=atime,delete=ctime",
		"--clamp-mtime",
		"--mtime=@0",
		"--numeric-owner",
		"--owner=0",
		"--group=0",
		"--mode=go+u,go-w",
		"-cf",
		tarballPath,
		"-C",
		filepath.Join(srcDir, ".."),
		filepath.Base(srcDir),
	)

	if err != nil {
		span.SetStatus(codes.Error, "error creating tarball")
		span.RecordError(err)
		return "", err
	}

	span.SetStatus(codes.Ok, "")
	span.RecordError(nil)
	return tarballPath, nil
}

func checkoutRef(ctx context.Context, repoPath string, ref string) error {
	ctx, span := tracer.Start(ctx, "checkoutRef", trace.WithAttributes(
		attribute.String("repoPath", repoPath),
		attribute.String("ref", ref),
	))
	defer span.End()

	span.AddEvent("checking out ref")
	_, _, err := command(ctx, "", []int{}, "git", "-C", repoPath, "checkout", ref)

	if err != nil {
		span.SetStatus(codes.Error, "failed to checkout ref")
		span.RecordError(err)
		return err
	}

	span.SetStatus(codes.Ok, "")
	span.RecordError(nil)
	return nil
}

func refToCommit(ctx context.Context, repoPath string, ref string) (string, error) {
	ctx, span := tracer.Start(ctx, "refToCommit", trace.WithAttributes(
		attribute.String("repoPath", repoPath),
		attribute.String("ref", ref),
	))
	defer span.End()

	span.AddEvent("getting commit from ref")
	stdout, _, err := command(ctx, "", []int{}, "git", "-C", repoPath, "rev-list", "-n", "1", ref)

	if err != nil {
		span.SetStatus(codes.Error, "failed to get commit from ref")
		span.RecordError(err)
		return "", err
	}

	span.SetStatus(codes.Ok, "")
	span.RecordError(nil)
	return strings.TrimSpace(stdout), nil
}

// Uses relies on git index
func generateDiff(
	ctx context.Context,
	workDir, headRepoPath, headRef, baseRef, diffFileName string,
	excludedPaths []string,
) (string, error) {
	ctx, span := tracer.Start(ctx, "generateDiff", trace.WithAttributes(
		attribute.String("headRepoPath", headRepoPath),
		attribute.String("baseRef", baseRef),
		attribute.String("diffFileName", diffFileName),
		attribute.String("workDir", workDir),
		attribute.StringSlice("excludedPaths", excludedPaths),
	))
	defer span.End()

	err := checkoutRef(ctx, headRepoPath, baseRef)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to checkout base ref")
		return "", err
	}
	err = checkoutRef(ctx, headRepoPath, headRef)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to checkout head ref")
		return "", err
	}

	args := []string{"-C", headRepoPath, "diff", "--binary", "--no-color", baseRef}

	args = append(args, "--")
	for _, excluded := range excludedPaths {
		args = append(args, fmt.Sprintf(":(exclude)%s", excluded))
	}

	stdout, _, err := command(
		ctx,
		"",
		[]int{1},
		"git",
		args...,
	)
	if err != nil {
		span.SetStatus(codes.Error, "error diffing")
		span.RecordError(err)
		return "", err
	}

	cleanDiff := stdout

	// Package in its own dir first
	diffOutDir := path.Join(workDir, "/diff")
	span.SetAttributes(attribute.String("diffDir", diffOutDir))
	span.AddEvent("creating diff dir")
	err = os.Mkdir(diffOutDir, 0700)
	if err != nil {
		span.SetStatus(codes.Error, "error creating diff dir")
		span.RecordError(err)
		return "", err
	}

	diffFilePath := path.Join(diffOutDir, diffFileName)
	span.SetAttributes(attribute.String("diffFile", diffFilePath))
	span.AddEvent("creating diff file")
	file, err := os.Create(diffFilePath)
	if err != nil {
		span.SetStatus(codes.Error, "error creating diff file")
		span.RecordError(err)
		return "", err
	}
	defer file.Close()

	span.AddEvent("writing diff to file")
	_, err = file.WriteString(cleanDiff)
	if err != nil {
		span.SetStatus(codes.Error, "error writing diff to file")
		span.RecordError(err)
		return "", err
	}

	span.SetStatus(codes.Ok, "")
	span.RecordError(nil)
	return diffOutDir, nil
}

func (h *Client) sendTask(
	ctx context.Context,
	roundID string,
	taskID string,
	messageID string,
	deadline time.Time,
	curlPayload []byte,
	teams []config.Team,
) {
	span := trace.SpanFromContext(ctx)

	// data should already be marshaled as a byte array at this point
	// So we just make our request
	span.AddEvent("sending task")
	h.send(ctx, "/v1/task/", roundID, taskID, messageID, deadline, curlPayload, teams)
}

// execute sarif broadcast or signal readiness (log it)
func (h *Client) SendSARIFBroadcast(
	ctx context.Context,
	roundID string,
	taskID string,
	messageID string,
	deadline time.Time,
	sarifPayload []byte,
	teams []config.Team,
) {
	span := trace.SpanFromContext(ctx)

	// data should already be marshaled as a byte array at this point
	// So we just make our request
	span.AddEvent("sending SARIF broadcast")
	h.send(ctx, "/v1/sarif/", roundID, taskID, messageID, deadline, sarifPayload, teams)
}

func (h *Client) send(
	ctx context.Context,
	route string,
	roundID string,
	taskID string,
	messageID string,
	deadline time.Time,
	payload []byte,
	teams []config.Team,
) {
	ctx, span := tracer.Start(ctx, "send")
	defer span.End()

	var teamIDs []string

	for _, team := range teams {
		teamIDs = append(teamIDs, team.ID)
	}

	span.SetAttributes(
		attribute.String("route", route),
		attribute.String("round.id", roundID),
		attribute.String("task.id", taskID),
		attribute.String("message.id", messageID),
		attribute.Int64("deadline_ms", deadline.UnixMilli()),
		attribute.Int("teamCount", len(teams)),
		attribute.StringSlice("teams", teamIDs),
	)

	span.AddEvent("create delivery job")
	_, err := h.jobClient.CreateDeliveryJob(
		ctx,
		route,
		teams,
		roundID,
		taskID,
		messageID,
		deadline,
		payload,
	)
	if err != nil {
		span.SetStatus(codes.Error, "failed to queue delivery job")
		span.RecordError(err)
	}
}

func (h *Client) RunFullScan(
	ctx context.Context,
	challengeInputs ChallengeConfig,
	teams []config.Team,
	roundID string,
) error {
	ctx, span := tracer.Start(ctx, "RunFullScan")
	defer span.End()

	db := h.db.WithContext(ctx)

	span.SetAttributes(
		attribute.String("challenge.name", challengeInputs.Name),
		attribute.Int64("challenge.duration_s", int64(challengeInputs.TaskDuration/time.Second)),
		attribute.String("challenge.repo.url", challengeInputs.RepoURL),
		attribute.String("challenge.refs.head", challengeInputs.HeadRef),
	)

	tarsDir := "tarsDir"
	stripRepoFiles := []string{".aixcc", ".github", ".git", ".gitattributes"}

	workDir, err := setupWorkspace(ctx, h.tempDir, tarsDir)
	if err != nil {
		span.SetStatus(codes.Error, "failed to setup workspace")
		span.RecordError(err)
		return err
	}
	defer os.RemoveAll(workDir)

	tarsDir = path.Join(workDir, tarsDir)

	repoPath, err := downloadRepo(ctx, challengeInputs.RepoURL, workDir, challengeInputs.AuthMethod)
	if err != nil {
		span.SetStatus(codes.Error, "failed to download repo")
		span.RecordError(err)
		return err
	}

	err = checkoutRef(ctx, repoPath, challengeInputs.HeadRef)
	if err != nil {
		span.SetStatus(codes.Error, "failed to checkout head ref")
		span.RecordError(err)
		return err
	}

	headCommit, err := refToCommit(ctx, repoPath, challengeInputs.HeadRef)
	if err != nil {
		span.SetStatus(codes.Error, "failed to convert ref to commit")
		span.RecordError(err)
		return err
	}

	challengeYAML, err := types.ParseChallengeYAML(ctx, repoPath)
	if err != nil {
		span.SetStatus(codes.Error, "failed to parse challenge YAML")
		span.RecordError(err)
		return err
	}

	harnessesIncluded := len(challengeYAML.HarnessesList) > 0

	err = setRepoURL(ctx, repoPath, challengeInputs.RepoURL, nil)
	if err != nil {
		span.SetStatus(codes.Error, "failed to set origin to unauthenticated URL")
		span.RecordError(err)
		return err
	}
	unstrippedRepoTarPath, err := packageRepo(ctx, repoPath, tarsDir)
	if err != nil {
		span.SetStatus(codes.Error, "failed to package unstripped repo")
		span.RecordError(err)
		return err
	}

	err = stripRepo(ctx, stripRepoFiles, repoPath)
	if err != nil {
		span.SetStatus(codes.Error, "failed to strip repo")
		span.RecordError(err)
		return err
	}

	strippedRepoTarPath, err := packageRepo(ctx, repoPath, tarsDir)
	if err != nil {
		span.SetStatus(codes.Error, "failed to package stripped repo")
		span.RecordError(err)
		return err
	}

	ossFuzzPath, err := downloadRepo(
		ctx,
		challengeYAML.FuzzToolingURL,
		workDir,
		challengeInputs.AuthMethod,
	)
	if err != nil {
		span.SetStatus(codes.Error, "failed to download fuzz tooling")
		span.RecordError(err)
		return err
	}

	err = checkoutRef(ctx, ossFuzzPath, challengeYAML.FuzzToolingRef)
	if err != nil {
		span.SetStatus(codes.Error, "failed to checkout fuzz tooling ref")
		span.RecordError(err)
		return err
	}

	err = stripRepo(ctx, stripRepoFiles, ossFuzzPath)
	if err != nil {
		span.SetStatus(codes.Error, "failed to strip fuzz tooling repo")
		span.RecordError(err)
		return err
	}

	span.AddEvent("rename oss-fuzz directory to fuzz-tooling")
	newOssFuzzPath := path.Join(path.Dir(ossFuzzPath), "fuzz-tooling")
	err = os.Rename(ossFuzzPath, newOssFuzzPath)
	if err != nil {
		span.SetStatus(codes.Error, "failed to rename oss-fuzz directory")
		span.RecordError(err)
		return err
	}
	ossFuzzPath = newOssFuzzPath

	ossFuzzTarPath, err := packageRepo(ctx, ossFuzzPath, tarsDir)
	if err != nil {
		span.SetStatus(codes.Error, "failed to package fuzz tooling repo")
		span.RecordError(err)
		return err
	}

	unstrippedRepoFileHash, err := upload.HashedFile(
		ctx,
		h.workingStore,
		unstrippedRepoTarPath,
	)

	if err != nil {
		span.SetStatus(codes.Error, "failed to get unstripped repo tarball hash")
		span.RecordError(err)
		return err
	}

	strippedRepoFileHash, err := upload.HashedFile(ctx, h.workingStore, strippedRepoTarPath)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get stripped repo tarball hash")
		span.RecordError(err)
		return err
	}

	ossFuzzFileHash, err := upload.HashedFile(ctx, h.workingStore, ossFuzzTarPath)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get fuzz tooling repo tarball hash")
		span.RecordError(err)
		return err
	}

	task := models.Task{
		Type:     types.TaskTypeFull,
		Deadline: time.Now().Add(challengeInputs.TaskDuration),
		RoundID:  roundID,
		Commit:   headCommit,
		MemoryGB: challengeYAML.MemoryGB,
		// baseRepoPath = tmp/base-repos/foobar
		// base = foobar
		Focus:             path.Base(repoPath),
		ProjectName:       challengeYAML.FuzzToolingProjectName,
		HarnessesIncluded: harnessesIncluded,
		Source: []models.Source{
			{
				Type:   string(types.SourceTypeRepo),
				URL:    strippedRepoFileHash,
				SHA256: strippedRepoFileHash,
			},
			{
				Type:   string(types.SourceTypeFuzzTooling),
				URL:    ossFuzzFileHash,
				SHA256: ossFuzzFileHash,
			},
		},
		UnstrippedSource: models.UnstrippedSources{
			HeadRepo: models.Source{
				Type:   string(types.SourceTypeRepo),
				URL:    unstrippedRepoFileHash,
				SHA256: unstrippedRepoFileHash,
			},
			FuzzTooling: models.Source{
				Type:   string(types.SourceTypeFuzzTooling),
				URL:    ossFuzzFileHash,
				SHA256: ossFuzzFileHash,
			},
		},
	}

	span.AddEvent("inserting task into db")
	err = db.Create(&task).Error
	if err != nil {
		span.SetStatus(codes.Error, "failed to insert task into db")
		span.RecordError(err)
		return err
	}
	taskID := task.ID.String()
	span.SetAttributes(attribute.String("task.id", taskID))

	expiration := task.Deadline.Add(time.Minute * 5)
	span.SetAttributes(attribute.Int64("presignedURLExpiration_ms", expiration.UnixMilli()))

	strippedRepoURL, err := h.workingStore.PresignedReadURL(
		ctx,
		strippedRepoFileHash,
		time.Until(expiration),
	)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get presigned url for stripped repo")
		span.RecordError(err)
		return err
	}

	fuzzToolingURL, err := h.workingStore.PresignedReadURL(
		ctx,
		ossFuzzFileHash,
		time.Until(expiration),
	)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get presigned url for fuzz tooling")
		span.RecordError(err)
		return err
	}

	auditContext := audit.Context{RoundID: roundID, TaskID: &taskID}
	files := []*archive.FileMetadata{
		{
			LocalFilePath: &unstrippedRepoTarPath,
			ArchivedFile:  types.FileUnstrippedRepoTarball,
			Entity:        audit.EntityTask,
			EntityID:      taskID,
		},
		{
			LocalFilePath: &strippedRepoTarPath,
			ArchivedFile:  types.FileStrippedRepoTarball,
			Entity:        audit.EntityTask,
			EntityID:      taskID,
		},
		{
			LocalFilePath: &ossFuzzTarPath,
			ArchivedFile:  types.FileOSSFuzzTarball,
			Entity:        audit.EntityTask,
			EntityID:      taskID,
		},
	}

	for _, f := range files {
		err = archive.ArchiveFile(ctx, auditContext, h.archiver, f)
		if err != nil {
			span.SetStatus(codes.Error, "failed to archive file")
			span.RecordError(err)
			return err
		}
	}

	span.AddEvent("generating audit log message")
	audit.LogNewFullScan(
		auditContext,
		challengeInputs.RepoURL,
		headCommit,
		types.UnixMilli(task.Deadline.UTC().UnixMilli()),
		challengeYAML.FuzzToolingURL,
		ossFuzzFileHash,
		challengeInputs.Name,
		!harnessesIncluded,
	)

	body := types.Task{
		MessageID:   uuid.New().String(),
		MessageTime: types.UnixMilli(time.Now().UTC().UnixMilli()),
		Tasks: []types.TaskDetail{
			{
				TaskID:            task.ID.String(),
				Type:              task.Type,
				Deadline:          types.UnixMilli(task.Deadline.UTC().UnixMilli()),
				ProjectName:       challengeYAML.FuzzToolingProjectName,
				Focus:             task.Focus,
				HarnessesIncluded: harnessesIncluded,
				Source: []types.SourceDetail{
					{
						Type:   types.SourceTypeRepo,
						URL:    strippedRepoURL,
						SHA256: strippedRepoFileHash,
					},
					{
						Type:   types.SourceTypeFuzzTooling,
						URL:    fuzzToolingURL,
						SHA256: ossFuzzFileHash,
					},
				},
				Metadata: types.TaskMetadata{
					TaskID:  task.ID.String(),
					RoundID: roundID,
				},
			},
		},
	}

	span.AddEvent("marshaling message as JSON")
	payload, err := json.Marshal(body)
	if err != nil {
		span.SetStatus(codes.Error, "failed to marshal message as JSON")
		span.RecordError(err)
		return err
	}

	h.sendTask(
		ctx,
		task.RoundID,
		task.ID.String(),
		body.MessageID,
		task.Deadline.UTC(),
		payload,
		teams,
	)

	span.SetStatus(codes.Ok, "")
	span.RecordError(nil)
	return nil
}

func (h *Client) RunDeltaScan(
	ctx context.Context,
	challengeInputs ChallengeConfig,
	teams []config.Team,
	roundID string,
) error {
	ctx, span := tracer.Start(ctx, "RunDeltaScan")
	defer span.End()

	db := h.db.WithContext(ctx)

	span.SetAttributes(
		attribute.String("challenge.name", challengeInputs.Name),
		attribute.Int64("challenge.duration_s", int64(challengeInputs.TaskDuration/time.Second)),
		attribute.String("challenge.repo.url", challengeInputs.RepoURL),
		attribute.String("challenge.refs.head", challengeInputs.HeadRef),
		attribute.String("challenge.refs.base", *challengeInputs.BaseRef),
	)

	tarsDir := "tarsDir"
	baseReposOuterDir := "baseRepos"
	stripRepoFiles := []string{".aixcc", ".github", ".git", ".gitattributes"}

	workDir, err := setupWorkspace(ctx, h.tempDir, tarsDir)
	if err != nil {
		span.SetStatus(codes.Error, "failed to setup workspace")
		span.RecordError(err)
		return err
	}
	defer os.RemoveAll(workDir)

	tarsDir = path.Join(workDir, tarsDir)
	baseReposOuterDir = path.Join(workDir, baseReposOuterDir)

	headRepoPath, err := downloadRepo(
		ctx,
		challengeInputs.RepoURL,
		workDir,
		challengeInputs.AuthMethod,
	)
	if err != nil {
		span.SetStatus(codes.Error, "Failed to download repo to head repo path")
		span.RecordError(err)
		return err
	}

	// headRepoPath = /tmp/dir/foobar
	// baseRepoPath = /tmp/dir/base-repos/foobar
	baseRepoPath := path.Join(baseReposOuterDir, path.Base(headRepoPath))
	span.SetAttributes(
		attribute.String("baseRepoPath", baseRepoPath),
		attribute.String("headRepoPath", headRepoPath),
	)

	span.AddEvent("creating repo dirs")
	err = os.MkdirAll(baseRepoPath, 0700)
	if err != nil {
		span.SetStatus(codes.Error, "failed to create baseRepoPath")
		span.RecordError(err)
		return err
	}

	span.AddEvent("copying repo to baseRepoPath")
	err = cp.Copy(headRepoPath, baseRepoPath, cp.Options{})
	if err != nil {
		span.SetStatus(codes.Error, "failed to copy repo")
		span.RecordError(err)
		return err
	}

	err = checkoutRef(ctx, headRepoPath, challengeInputs.HeadRef)
	if err != nil {
		span.SetStatus(codes.Error, "failed to checkout head ref in head repo path")
		span.RecordError(err)
		return err
	}

	headCommit, err := refToCommit(ctx, headRepoPath, challengeInputs.HeadRef)
	if err != nil {
		span.SetStatus(codes.Error, "failed to convert head ref to commit")
		span.RecordError(err)
		return err
	}

	challengeYAML, err := types.ParseChallengeYAML(ctx, headRepoPath)
	if err != nil {
		span.SetStatus(codes.Error, "failed to parse challenge YAML")
		span.RecordError(err)
		return err
	}

	harnessesIncluded := len(challengeYAML.HarnessesList) > 0

	err = checkoutRef(ctx, baseRepoPath, *challengeInputs.BaseRef)
	if err != nil {
		span.SetStatus(codes.Error, "failed to checkout base ref in base repo path")
		span.RecordError(err)
		return err
	}

	baseCommit, err := refToCommit(ctx, baseRepoPath, *challengeInputs.BaseRef)
	if err != nil {
		span.SetStatus(codes.Error, "failed to convert base ref to commit")
		span.RecordError(err)
		return err
	}

	eg, egctx := errgroup.WithContext(ctx)
	var unstrippedHeadTarPath string
	eg.Go(func() error {
		fail := setRepoURL(ctx, headRepoPath, challengeInputs.RepoURL, nil)
		if err != nil {
			return fail
		}
		uhtp, fail := packageRepo(egctx, headRepoPath, tarsDir)
		if fail != nil {
			return fail
		}
		unstrippedHeadTarPath = uhtp
		return nil
	})

	var unstrippedBaseTarPath string
	eg.Go(func() error {
		fail := setRepoURL(ctx, baseRepoPath, challengeInputs.RepoURL, nil)
		if err != nil {
			return fail
		}
		ubtp, fail := packageRepo(egctx, baseRepoPath, tarsDir)
		if fail != nil {
			return fail
		}
		unstrippedBaseTarPath = ubtp
		return nil
	})
	err = eg.Wait()
	if err != nil {
		span.SetStatus(codes.Error, "failed to package unstripped repos")
		span.RecordError(err)
		return err
	}

	diffPath, err := generateDiff(
		ctx,
		workDir,
		headRepoPath,
		challengeInputs.HeadRef,
		*challengeInputs.BaseRef,
		"ref.diff",
		stripRepoFiles,
	)
	if err != nil {
		span.SetStatus(codes.Error, "failed to generate diff")
		span.RecordError(err)
		return err
	}

	ossFuzzPath, err := downloadRepo(
		ctx,
		challengeYAML.FuzzToolingURL,
		workDir,
		challengeInputs.AuthMethod,
	)
	if err != nil {
		span.SetStatus(codes.Error, "failed to download fuzz tooling repo")
		span.RecordError(err)
		return err
	}

	err = checkoutRef(ctx, ossFuzzPath, challengeYAML.FuzzToolingRef)
	if err != nil {
		span.SetStatus(codes.Error, "failed to checkout fuzz tooling ref")
		span.RecordError(err)
		return err
	}

	err = stripRepo(ctx, stripRepoFiles, baseRepoPath)
	if err != nil {
		span.SetStatus(codes.Error, "failed to strip base repo")
		span.RecordError(err)
		return err
	}

	err = stripRepo(ctx, stripRepoFiles, ossFuzzPath)
	if err != nil {
		span.SetStatus(codes.Error, "failed to strip fuzz tooling repo")
		span.RecordError(err)
		return err
	}

	eg, egctx = errgroup.WithContext(ctx)
	var strippedRepoTarPath string
	eg.Go(func() error {
		srtp, fail := packageRepo(egctx, baseRepoPath, tarsDir)
		if fail != nil {
			return fail
		}
		strippedRepoTarPath = srtp
		return nil
	})
	var ossFuzzTarPath string
	eg.Go(func() error {
		span.AddEvent("rename oss-fuzz directory to fuzz-tooling")
		newOssFuzzPath := path.Join(path.Dir(ossFuzzPath), "fuzz-tooling")
		fail := os.Rename(ossFuzzPath, newOssFuzzPath)
		if fail != nil {
			span.SetStatus(codes.Error, "failed to rename oss-fuzz directory")
			span.RecordError(fail)
			return fail
		}
		ossFuzzPath = newOssFuzzPath

		oftp, fail := packageRepo(egctx, ossFuzzPath, tarsDir)
		if fail != nil {
			return fail
		}
		ossFuzzTarPath = oftp
		return nil
	})
	var diffTarPath string
	eg.Go(func() error {
		dtp, fail := packageRepo(egctx, diffPath, tarsDir)
		if fail != nil {
			return fail
		}
		diffTarPath = dtp
		return nil
	})
	err = eg.Wait()
	if err != nil {
		span.SetStatus(codes.Error, "failed to package repos")
		span.RecordError(err)
		return err
	}

	unstrippedHeadFileHash, err := upload.HashedFile(
		ctx,
		h.workingStore,
		unstrippedHeadTarPath,
	)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get unstripped head repo tarball hash")
		span.RecordError(err)
		return err
	}

	unstrippedBaseFileHash, err := upload.HashedFile(
		ctx,
		h.workingStore,
		unstrippedBaseTarPath,
	)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get unstripped base repo tarball hash")
		span.RecordError(err)
		return err
	}

	strippedRepoFileHash, err := upload.HashedFile(ctx, h.workingStore, strippedRepoTarPath)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get stripped base repo tarball hash")
		span.RecordError(err)
		return err
	}

	ossFuzzFileHash, err := upload.HashedFile(ctx, h.workingStore, ossFuzzTarPath)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get fuzz tooling repo tarball hash")
		span.RecordError(err)
		return err
	}

	diffFileHash, err := upload.HashedFile(ctx, h.workingStore, diffTarPath)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get diff tarball hash")
		span.RecordError(err)
		return err
	}

	task := models.Task{
		Type:     types.TaskTypeDelta,
		Deadline: time.Now().Add(challengeInputs.TaskDuration),
		RoundID:  roundID,
		Commit:   headCommit,
		MemoryGB: challengeYAML.MemoryGB,
		// baseRepoPath = tmp/base-repos/foobar
		// base = foobar
		Focus:             path.Base(baseRepoPath),
		ProjectName:       challengeYAML.FuzzToolingProjectName,
		HarnessesIncluded: harnessesIncluded,
		Source: []models.Source{
			{
				Type:   string(types.SourceTypeRepo),
				URL:    strippedRepoFileHash,
				SHA256: strippedRepoFileHash,
			},
			{
				Type:   string(types.SourceTypeFuzzTooling),
				URL:    ossFuzzFileHash,
				SHA256: ossFuzzFileHash,
			},
			{
				Type:   string(types.SourceTypeDiff),
				URL:    diffFileHash,
				SHA256: diffFileHash,
			},
		},
		UnstrippedSource: models.UnstrippedSources{
			HeadRepo: models.Source{
				Type:   string(types.SourceTypeRepo),
				URL:    unstrippedHeadFileHash,
				SHA256: unstrippedHeadFileHash,
			},
			FuzzTooling: models.Source{
				Type:   string(types.SourceTypeFuzzTooling),
				URL:    ossFuzzFileHash,
				SHA256: ossFuzzFileHash,
			},
			BaseRepo: &models.Source{
				Type:   string(types.SourceTypeRepo),
				URL:    unstrippedBaseFileHash,
				SHA256: unstrippedBaseFileHash,
			},
		},
	}

	span.AddEvent("inserting task into db")
	err = db.Create(&task).Error
	if err != nil {
		span.SetStatus(codes.Error, "failed to insert task into db")
		span.RecordError(err)
		return err
	}
	taskID := task.ID.String()
	span.SetAttributes(attribute.String("task.id", taskID))

	expiration := task.Deadline.Add(time.Minute * 5)
	span.SetAttributes(attribute.Int64("presignedURLExpiration_ms", expiration.UnixMilli()))

	strippedRepoURL, err := h.workingStore.PresignedReadURL(
		ctx,
		strippedRepoFileHash,
		time.Until(expiration),
	)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get presigned url for stripped repo")
		span.RecordError(err)
		return err
	}

	fuzzToolingURL, err := h.workingStore.PresignedReadURL(
		ctx,
		ossFuzzFileHash,
		time.Until(expiration),
	)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get presigned url for fuzz tooling repo")
		span.RecordError(err)
		return err
	}

	diffURL, err := h.workingStore.PresignedReadURL(ctx, diffFileHash, time.Until(expiration))
	if err != nil {
		span.SetStatus(codes.Error, "failed to get presigned url for diff")
		span.RecordError(err)
		return err
	}

	auditContext := audit.Context{RoundID: roundID, TaskID: &taskID}
	files := []*archive.FileMetadata{
		{
			LocalFilePath: &unstrippedHeadTarPath,
			ArchivedFile:  types.FileUnstrippedHeadTarball,
			Entity:        audit.EntityTask,
			EntityID:      taskID,
		},
		{
			LocalFilePath: &unstrippedBaseTarPath,
			ArchivedFile:  types.FileUnstrippedBaseTarball,
			Entity:        audit.EntityTask,
			EntityID:      taskID,
		},
		{
			LocalFilePath: &diffTarPath,
			ArchivedFile:  types.FileDiffTarball,
			Entity:        audit.EntityTask,
			EntityID:      taskID,
		},
		{
			LocalFilePath: &strippedRepoTarPath,
			ArchivedFile:  types.FileStrippedRepoTarball,
			Entity:        audit.EntityTask,
			EntityID:      taskID,
		},
		{
			LocalFilePath: &ossFuzzTarPath,
			ArchivedFile:  types.FileOSSFuzzTarball,
			Entity:        audit.EntityTask,
			EntityID:      taskID,
		},
	}
	for _, f := range files {
		err = archive.ArchiveFile(ctx, auditContext, h.archiver, f)
		if err != nil {
			span.SetStatus(codes.Error, "failed to archive file")
			span.RecordError(err)
			return err
		}
	}

	span.AddEvent("generating audit log message")
	audit.LogNewDeltaScan(
		auditContext,
		challengeInputs.RepoURL,
		baseCommit,
		headCommit,
		types.UnixMilli(task.Deadline.UTC().UnixMilli()),
		challengeYAML.FuzzToolingURL,
		ossFuzzFileHash,
		challengeInputs.Name,
		!harnessesIncluded,
	)

	body := types.Task{
		MessageID:   uuid.New().String(),
		MessageTime: types.UnixMilli(time.Now().UTC().UnixMilli()),
		Tasks: []types.TaskDetail{
			{
				TaskID:            task.ID.String(),
				Type:              task.Type,
				Deadline:          types.UnixMilli(task.Deadline.UTC().UnixMilli()),
				ProjectName:       task.ProjectName,
				Focus:             task.Focus,
				HarnessesIncluded: harnessesIncluded,
				Source: []types.SourceDetail{
					{
						Type:   types.SourceTypeRepo,
						URL:    strippedRepoURL,
						SHA256: strippedRepoFileHash,
					},
					{
						Type:   types.SourceTypeFuzzTooling,
						URL:    fuzzToolingURL,
						SHA256: ossFuzzFileHash,
					},
					{
						Type:   types.SourceTypeDiff,
						URL:    diffURL,
						SHA256: diffFileHash,
					},
				},
				Metadata: types.TaskMetadata{
					TaskID:  task.ID.String(),
					RoundID: roundID,
				},
			},
		},
	}

	span.AddEvent("marshaling message as JSON")
	payload, err := json.Marshal(body)
	if err != nil {
		span.SetStatus(codes.Error, "failed to marshal message as JSON")
		span.RecordError(err)
		return err
	}

	h.sendTask(
		ctx,
		task.RoundID,
		task.ID.String(),
		body.MessageID,
		task.Deadline.UTC(),
		payload,
		teams,
	)

	span.SetStatus(codes.Ok, "")
	span.RecordError(nil)
	return nil
}

func replaceSSH(sshURL string) string {
	return strings.Replace(sshURL, "git@github.com:", "https://github.com/", 1)
}
