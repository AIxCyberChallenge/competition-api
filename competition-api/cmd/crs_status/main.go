package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/config"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

func main() {
	logger.InitSlog()

	conf, err := config.GetConfig()
	if err != nil {
		logger.Logger.Error("error calling GetConfig", "error", err)
		os.Exit(1)
	}

	logger.LogLevel.Set(slog.Level(conf.Logging.App.Level))

	l := logger.Logger.WithGroup("statusCheck")

	waitTime := time.Duration(*conf.CRSStatusPollTimeSeconds) * time.Second

	httpClient := retryablehttp.NewClient()
	httpClient.HTTPClient.Timeout = waitTime
	httpClient.RetryMax = 3

	for {
		wg := sync.WaitGroup{}
		errChan := make([]error, len(conf.Teams))

		for count, team := range conf.Teams {
			if team.CRS == nil {
				l.Debug("skipping team with no configured CRS", "team", team.ID, "note", team.Note)
				continue
			}

			auditContext := audit.Context{
				TeamID:  &team.ID,
				RoundID: *conf.RoundID,
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				status, fail := checkStatus(l, httpClient.StandardClient(), team)
				if fail != nil {
					audit.LogCRSStatusFailed(auditContext, team.CRS.URL, fail.Error())
					errChan[count] = fail
					return
				}

				audit.LogCRSStatus(auditContext, team.CRS.URL, status)
			}()
		}
		wg.Wait()
		for _, err := range errChan {
			if err != nil {
				l.Error(err.Error())
			}
		}
		time.Sleep(waitTime)
	}
}

func checkStatus(l *slog.Logger, httpClient *http.Client, team config.Team) (*types.Status, error) {
	l = l.With("crs", team.ID)

	req, err := http.NewRequest(http.MethodGet, team.CRS.URL+"/status/", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to construct status request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	req.SetBasicAuth(team.CRS.APIKeyID, team.CRS.APIKeyToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		l.Error("failed to send CRS status request", "error", err)
		return nil, fmt.Errorf("failed to send crs status request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		l.Error("invalid status code", "code", resp.StatusCode)
		return nil, fmt.Errorf("invalid status code: %v", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		l.Error("error read response body")
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	respData := types.Status{}

	err = json.Unmarshal(body, &respData)
	if err != nil {
		l.Error("error unmarshaling json Status data from body")
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	l.Debug("Event payload", "payload", respData)

	return &respData, nil
}
