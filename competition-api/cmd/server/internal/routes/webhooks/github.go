package webhooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v66/github"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/challenges"
	srverr "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/error"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/response"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/archive"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/validator"
)

const name = "github.com/aixcyberchallenge/competition-api/competition-api/server/routes/competition/webhooks"

var tracer = otel.Tracer(name)

type Reaction string

const (
	ReactionPositive = "+1"
	ReactionNegative = "eyes"
)

type taskOptions struct {
	DurationSecs *int64 `json:"duration_secs" validate:"required"`
}

type githubObjectDescription struct {
	ChallengeName *string      `json:"challenge_name" validate:"required"`
	TaskOpts      *taskOptions `json:"task_opts"      validate:"required"`
}

func React(ctx context.Context, token *github.InstallationToken, event any, react Reaction) error {
	ctx, span := tracer.Start(ctx, "React")
	defer span.End()

	span.SetAttributes(attribute.String("reaction", string(react)))

	span.AddEvent("adding reaction")
	var err error
	switch e := event.(type) {
	case *github.PullRequestEvent:
		_, _, err = github.NewClient(nil).WithAuthToken(token.GetToken()).Reactions.CreateIssueReaction(ctx, e.GetRepo().GetOwner().GetLogin(), e.GetRepo().GetName(), e.GetNumber(), string(react))
		if err != nil {
			span.SetStatus(codes.Error, "failed to set status indicator")
			span.RecordError(err)
			return err
		}
		span.SetStatus(codes.Ok, "")
		span.RecordError(nil)
		return nil
	case *github.ReleaseEvent:
		_, _, err = github.NewClient(nil).WithAuthToken(token.GetToken()).Reactions.CreateReleaseReaction(ctx, e.GetRepo().GetOwner().GetLogin(), e.GetRepo().GetName(), e.GetRelease().GetID(), string(react))
		if err != nil {
			span.SetStatus(codes.Error, "failed to set status indicator")
			span.RecordError(err)
			return err
		}
		span.SetStatus(codes.Ok, "")
		span.RecordError(nil)
		return nil
	}

	err = errors.New("tried to react to an unsupported event")
	span.RecordError(err)
	span.SetStatus(codes.Error, "tried to react to an unsupported event")
	return err
}

func (h *Handler) HandleGithubWebhook(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "HandleGithubWebhook")
	defer span.End()

	span.AddEvent("received GitHub webhook")

	requestTime, ok := c.Get("time").(time.Time)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("time: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	span.SetAttributes(attribute.Int64("request.timestamp_ms", requestTime.UnixMilli()))

	event, err := h.githubClient.ParseWebhookPayload(c.Request())
	if err != nil {
		span.SetStatus(codes.Error, "failed to parse GitHub webhook")
		span.RecordError(err)
		return c.JSON(http.StatusBadRequest, types.StringError("failed to parse webhook payload"))
	}

	// pre-switch checking

	switch e := event.(type) {
	case *github.PullRequestEvent:
		span.SetAttributes(
			attribute.String("webhook.action", e.GetAction()),
			attribute.String("webhook.pullRequest.title", e.GetPullRequest().GetTitle()),
		)

		span.AddEvent("validating webhook data")

		if e.Repo == nil || e.Repo.HTMLURL == nil {
			span.SetStatus(codes.Error, "webhook payload is missing repo URL")
			span.RecordError(errors.New("webhook payload is missing repo url"))
			return c.JSON(http.StatusBadRequest, types.StringError("failed to parse webhook payload"))
		}

		repoURL := e.Repo.GetHTMLURL()
		span.SetAttributes(attribute.String("webhook.repo.url", repoURL))

		if slices.Contains(*h.ignoredRepos, repoURL) {
			span.SetStatus(codes.Ok, "ignoring pull request since repo URL is in IgnoredRepos list")
			span.RecordError(nil)
			return nil
		}

		if e.GetAction() != "opened" && e.GetAction() != "reopened" {
			span.SetStatus(codes.Ok, "ignoring pull request event with an action which is not `opened` or `reopened`")
			span.RecordError(nil)
			return nil
		}

		baseRef := e.GetPullRequest().GetBase().GetRef()
		diffRef := e.GetPullRequest().GetHead().GetRef()
		span.SetAttributes(
			attribute.String("webhook.pullRequest.baseRef", baseRef),
			attribute.String("webhook.pullRequest.diffRef", diffRef),
		)

		installationID := e.GetInstallation().GetID()
		if installationID == 0 {
			err = errors.New("installationID field in GithubWebhook.Installation empty")
			span.RecordError(err)
			span.SetStatus(codes.Error, "installationID field in GithubWebhook.Installation empty")
			return err
		}

		span.SetAttributes(attribute.Int64("webhook.installation.id", installationID))

		h.taskRunnerClient.Run(ctx, func(ctx context.Context) {
			//nolint:govet // shadow: intentionally shadow ctx and span to avoid using the incorrect one.
			ctx, span := tracer.Start(ctx, "HandleGithubWebhookPullRequestTaskingFunction")
			defer span.End()

			span.AddEvent("get token for installation ID")
			token, err := h.githubClient.CreateInstallationToken(ctx, installationID)
			if err != nil {
				span.SetStatus(codes.Error, "failed to get installation token")
				span.RecordError(err)
				return
			}

			span.AddEvent("getting & parsing challenge description")
			challengeDescription, err := parseGithubTaskDescription(ctx, e.GetPullRequest().GetBody())
			if err != nil {
				span.SetStatus(codes.Error, "error getting & parsing challenge description")
				span.RecordError(err)
				return
			}

			duration := time.Duration(*challengeDescription.TaskOpts.DurationSecs) * time.Second
			challengeName := challengeDescription.ChallengeName
			span.SetAttributes(
				attribute.Int64("challenge.duration_s", int64(duration/time.Second)),
				attribute.String("challenge.name", *challengeName),
			)

			span.AddEvent("starting run delta for all teams")
			err = h.challengesClient.RunDeltaScan(ctx, challenges.ChallengeConfig{
				Name:         *challengeName,
				TaskDuration: duration,
				RepoURL:      repoURL,
				HeadRef:      diffRef,
				BaseRef:      &baseRef,
				AuthMethod: githttp.BasicAuth{
					Username: "arbitrary",
					Password: token.GetToken(),
				},
			}, h.teams, h.roundID)

			if err != nil {
				span.SetStatus(codes.Error, "failed to run delta scan")
				span.RecordError(err)

				err = React(ctx, token, e, ReactionNegative)
				if err != nil {
					span.AddEvent("failed to set status indicator")
				}
				return
			}

			err = React(ctx, token, e, ReactionPositive)
			if err != nil {
				span.SetStatus(codes.Error, "failed to set status indicator")
				span.RecordError(err)
			}
		})
	case *github.ReleaseEvent:
		span.SetAttributes(
			attribute.String("webhook.action", e.GetAction()),
			attribute.String("webhook.release.tag", e.GetRelease().GetTagName()),
		)

		span.AddEvent("validating webhook data")

		if e.Repo == nil || e.Repo.HTMLURL == nil {
			span.RecordError(nil)
			span.SetStatus(codes.Error, "webhook payload is missing repo URL")
			return c.JSON(http.StatusBadRequest, types.StringError("failed to parse webhook payload"))
		}

		repoURL := e.Repo.GetHTMLURL()
		span.SetAttributes(attribute.String("webhook.repo.url", repoURL))

		if slices.Contains(*h.ignoredRepos, repoURL) {
			span.SetStatus(codes.Ok, "ignoring pull request since repo URL is in IgnoredRepos list")
			span.RecordError(nil)
			return nil
		}

		if e.GetAction() != "published" {
			// A release, pre-release, or draft of a release
			span.SetStatus(codes.Ok, "ignoring release event with an action which is not `published`")
			span.RecordError(nil)
			return nil
		}

		installationID := e.GetInstallation().GetID()
		if installationID == 0 {
			err = errors.New("installationID field in GithubWebhook.Installation empty")
			span.RecordError(err)
			span.SetStatus(codes.Error, "installationID field in GithubWebhook.Installation empty")
			return err
		}

		span.SetAttributes(attribute.Int64("webhook.installation.id", installationID))

		h.taskRunnerClient.Run(ctx, func(ctx context.Context) {
			//nolint:govet // shadow: intentionally shadow ctx and span to avoid using the incorrect one.
			ctx, span := tracer.Start(ctx, "HandleGithubWebhookReleaseTaskingFunction")
			defer span.End()

			span.AddEvent("get token for installation ID")
			token, err := h.githubClient.CreateInstallationToken(ctx, installationID)
			if err != nil {
				span.SetStatus(codes.Error, "failed to get installation token")
				span.RecordError(err)
				return
			}

			repoURL := e.Repo.GetHTMLURL()

			span.AddEvent("getting & parsing challenge description")
			challengeDescription, err := parseGithubTaskDescription(ctx, e.GetRelease().GetBody())
			if err != nil {
				span.SetStatus(codes.Error, "error getting & parsing challenge description")
				span.RecordError(err)
				return
			}

			duration := time.Duration(*challengeDescription.TaskOpts.DurationSecs) * time.Second
			challengeName := challengeDescription.ChallengeName
			span.SetAttributes(
				attribute.Int64("challenge.duration_s", int64(duration/time.Second)),
				attribute.String("challenge.name", *challengeName),
			)

			err = h.challengesClient.RunFullScan(ctx, challenges.ChallengeConfig{
				Name:         *challengeName,
				TaskDuration: duration,
				RepoURL:      repoURL,
				HeadRef:      e.Release.GetTagName(),
				AuthMethod: githttp.BasicAuth{
					Username: "arbitrary",
					Password: token.GetToken(),
				},
			}, h.teams, h.roundID)

			if err != nil {
				span.SetStatus(codes.Error, "failed to run delta scan")
				span.RecordError(err)

				err = React(ctx, token, e, ReactionNegative)
				if err != nil {
					span.AddEvent("failed to set status indicator")
				}
				return
			}

			err = React(ctx, token, e, ReactionPositive)
			if err != nil {
				span.SetStatus(codes.Error, "failed to set status indicator")
				span.RecordError(err)
				return
			}
		})
	case *github.CodeScanningAlertEvent:
		action := e.GetAction()

		span.SetAttributes(
			attribute.String("webhook.action", action),
			attribute.String("sarifBroadcast.commit", e.GetCommitOID()),
		)

		span.AddEvent("validating webhook data")

		if e.Repo == nil || e.Repo.HTMLURL == nil {
			span.RecordError(nil)
			span.SetStatus(codes.Error, "webhook payload is missing repo URL")
			logger.Logger.ErrorContext(ctx, "code_alert_error")
			return c.JSON(http.StatusBadRequest, types.StringError("failed to parse webhook payload"))
		}

		repoURL := e.Repo.GetHTMLURL()
		span.SetAttributes(attribute.String("webhook.repo.url", repoURL))

		if slices.Contains(*h.ignoredRepos, repoURL) {
			span.AddEvent("ignoring pull request since repo URL is in IgnoredRepos list")
			span.SetStatus(codes.Ok, "ignoring pull request since repo URL is in IgnoredRepos list")
			span.RecordError(nil)
			return nil
		}

		if action != "created" {
			span.AddEvent("ignoring code scanning alert event with an action which is not `created`")
			span.SetStatus(codes.Ok, "ignoring code scanning event with an action which is not `created`")
			span.RecordError(nil)
			return nil
		}

		installationID := e.GetInstallation().GetID()
		if installationID == 0 {
			err = errors.New("installationID field in GithubWebhook.Installation empty")
			span.RecordError(err)
			span.SetStatus(codes.Error, "installationID field in GithubWebhook.Installation empty")
			logger.Logger.ErrorContext(ctx, "code_alert_error")
			return err
		}
		span.SetAttributes(attribute.Int64("webhook.installationID", installationID))

		h.taskRunnerClient.Run(ctx, func(ctx context.Context) {
			//nolint:govet // shadow: intentionally shadow ctx and span to avoid using the incorrect one.
			ctx, span := tracer.Start(ctx, "HandleGithubWebhookCodeScanningAlertTaskingFunction")
			defer span.End()

			db := h.db.WithContext(ctx)

			span.AddEvent("get token for installation ID")
			token, err := h.githubClient.CreateInstallationToken(ctx, installationID)
			if err != nil {
				span.SetStatus(codes.Error, "failed to get installation token")
				logger.Logger.ErrorContext(ctx, "code_alert_error")
				span.RecordError(err)
				return
			}

			span.AddEvent("getting newest task for commit")
			task := new(models.Task)
			result := db.Model(task).Where("commit = ? AND round_id = ? AND deadline > CURRENT_TIMESTAMP", e.GetCommitOID(), h.roundID).Order("created_at desc").First(task)
			if result.Error != nil {
				span.SetStatus(codes.Error, "error getting task from database")
				span.RecordError(err)
				logger.Logger.ErrorContext(ctx, "code_alert_error")
				return
			}
			span.SetAttributes(attribute.String("sarifBroadcast.matchedTask.id", task.ID.String()))

			span.AddEvent("checking if this task already has a SARIF broadcast")
			exists, err := models.Exists[models.SARIFBroadcast](ctx, db, "task_id = ?", task.ID)
			if err != nil {
				span.SetStatus(codes.Error, "failed to check if SARIF broadcast already exists")
				span.RecordError(err)
				logger.Logger.ErrorContext(ctx, "code_alert_error")
				return
			}
			if exists {
				span.SetStatus(codes.Error, "this task already has a SARIF broadcast.  ignoring this event")
				span.RecordError(nil)
				return
			}

			span.AddEvent("locating analysis ID for this code scanning alert")
			client := github.NewClient(nil).WithAuthToken(token.GetToken())
			page := 1
			var analyses []*github.ScanningAnalysis
			var r *github.Response
			var analysis *github.ScanningAnalysis

			owner := e.GetRepo().GetOwner().GetLogin()
			repo := e.GetRepo().GetName()
			span.SetAttributes(
				attribute.String("sarifBroadcast.owner", owner),
				attribute.String("sarifBroadcast.repo", repo),
			)

			if owner == "" || repo == "" {
				span.SetStatus(codes.Error, "missing repo or owner")
				span.RecordError(err)
				logger.Logger.ErrorContext(ctx, "code_alert_error")
				return
			}

			// find analysis id (thanks github)
		Outer:
			for {
				span.SetAttributes(attribute.Int("sarifBroadcast.analysisList.page", page))
				span.AddEvent("checking analysis list page")
				// there may be better custom params we could provide for filtering
				analyses, r, err = client.CodeScanning.ListAnalysesForRepo(ctx,
					owner,
					repo,
					&github.AnalysesListOptions{
						Ref:         e.Ref,
						ListOptions: github.ListOptions{PerPage: 100, Page: page},
					})
				if err != nil {
					span.SetStatus(codes.Error, "failed to get analysis page")
					span.RecordError(err)
					logger.Logger.ErrorContext(ctx, "code_alert_error")
					return
				}

				for _, analysisCandidate := range analyses {
					if analysisCandidate.GetCommitSHA() == e.GetCommitOID() {
						span.AddEvent("matched analysis commit SHA")
						analysis = analysisCandidate
						break Outer
					}
				}

				// we are on the last page give up
				if page == r.LastPage {
					span.RecordError(errors.New("failed to find analysis for code scanning alert"))
					span.SetStatus(codes.Error, "failed to find analysis for code scanning alert")
					logger.Logger.ErrorContext(ctx, "code_alert_error")
					return
				}

				// walk to the next page and check that
				page = r.NextPage
			}

			span.SetAttributes(attribute.String("sarifBroadcast.analysisURL", analysis.GetURL()))
			span.AddEvent("building request for SARIF report")
			req, err := client.NewRequest(http.MethodGet, analysis.GetURL(), nil)
			if err != nil {
				span.SetStatus(codes.Error, "failed to create request")
				span.RecordError(err)
				logger.Logger.ErrorContext(ctx, "code_alert_error")
				return
			}
			req.Header.Set("Accept", "application/sarif+json")
			sarif := datatypes.JSON{}
			span.AddEvent("getting SARIF report")
			_, err = client.Do(ctx, req, &sarif)
			if err != nil {
				span.SetStatus(codes.Error, "failed to get SARIF report")
				span.RecordError(err)
				logger.Logger.ErrorContext(ctx, "code_alert_error")
				return
			}

			internalSARIFBroadcast := &models.SARIFBroadcast{
				TaskID: task.ID,
				SARIF:  sarif,
			}

			span.AddEvent("inserting into database")
			result = db.Create(internalSARIFBroadcast)
			if result.Error != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					span.SetStatus(codes.Error, "already sent a sarif broadcast for this task")
					span.RecordError(errors.New("already sent a SARIF broadcast for this task"))
					return
				}

				span.SetStatus(codes.Error, "failed to insert")
				span.RecordError(err)
				return
			}

			taskID := internalSARIFBroadcast.TaskID.String()
			span.SetAttributes(
				attribute.String("task.id", taskID),
				attribute.String("sarifBroadcast.id", internalSARIFBroadcast.ID.String()),
			)

			crsSARIFBroadcast := types.SARIFBroadcast{
				MessageID:   uuid.New().String(),
				MessageTime: types.UnixMilli(time.Now().UTC().UnixMilli()),
				Broadcasts: []types.SARIFBroadcastDetail{
					{
						TaskID:  taskID,
						SARIFID: internalSARIFBroadcast.ID.String(),
						SARIF:   internalSARIFBroadcast.SARIF,
						Metadata: types.SARIFBroadcastMetadata{
							TaskID:  internalSARIFBroadcast.TaskID.String(),
							SARIFID: internalSARIFBroadcast.ID.String(),
							RoundID: task.RoundID,
						},
					},
				},
			}

			auditContext := audit.Context{RoundID: task.RoundID, TaskID: &taskID}
			span.AddEvent("encoding SARIF broadcast as JSON")
			rawSARIF, err := json.Marshal(internalSARIFBroadcast.SARIF)
			if err != nil {
				span.SetStatus(codes.Error, "failed to encode SARIF broadcast")
				span.RecordError(err)
				logger.Logger.ErrorContext(ctx, "code_alert_error")
				return
			}

			upload := &archive.FileMetadata{
				Buffer:       &rawSARIF,
				ArchivedFile: types.FileSARIFBroadcast,
				Entity:       audit.EntityTask,
				EntityID:     taskID,
			}

			err = archive.ArchiveFile(ctx, auditContext, h.archiver, upload)
			if err != nil {
				span.SetStatus(codes.Error, "failed to archive file")
				span.RecordError(err)
				return
			}

			span.AddEvent("generating audit log message")
			audit.LogNewSARIFBroadcast(auditContext, e.Repo.GetHTMLURL(), e.CommitOID, internalSARIFBroadcast.ID.String())
			logger.Logger.WarnContext(ctx, "received_sarif_from_github")

			span.AddEvent("encoding SARIF broadcast event as JSON")
			crsPayload, err := json.Marshal(crsSARIFBroadcast)
			if err != nil {
				span.SetStatus(codes.Error, "failed to encode payload")
				span.RecordError(err)
				logger.Logger.ErrorContext(ctx, "code_alert_error")
				return
			}

			span.AddEvent("sending SARIF broadcast")
			h.challengesClient.SendSARIFBroadcast(ctx, task.RoundID, task.ID.String(), crsSARIFBroadcast.MessageID, task.Deadline.UTC(), crsPayload, h.teams)

			span.RecordError(nil)
			span.SetStatus(codes.Ok, "sent sarif broadcast")
		})
	default:
		span.RecordError(nil)
		span.SetStatus(codes.Ok, "ignoring unhandled event type")
		return nil
	}

	// Respond to GitHub
	span.SetStatus(codes.Ok, "successfully handled github webhook")
	span.RecordError(nil)
	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

func parseGithubTaskDescription(
	ctx context.Context,
	body string,
) (*githubObjectDescription, error) {
	_, span := tracer.Start(ctx, "parseGithubTaskDescription")
	defer span.End()

	span.SetAttributes(attribute.String("body", body))

	span.AddEvent("unmarshaling task body")
	taskDescription := new(githubObjectDescription)
	err := json.Unmarshal([]byte(body), taskDescription)
	if err != nil {
		span.SetStatus(codes.Error, "failed to unmarshal task body")
		span.RecordError(err)
		return nil, err
	}

	span.AddEvent("validating task body")
	v := validator.Create()
	err = v.Validate(taskDescription)
	if err != nil {
		span.SetStatus(codes.Error, "failed to validate task body")
		span.RecordError(err)
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	span.RecordError(nil)
	return taskDescription, nil
}
