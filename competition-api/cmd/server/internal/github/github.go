package github

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v66/github"
	"github.com/hashicorp/go-retryablehttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/config"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
)

const name = "github.com/aixcyberchallenge/competition-api/competition-api/github"

var tracer = otel.Tracer(name)

type Client struct {
	credentials *config.GithubConfig
	appClient   *github.Client
}

func Create(credentials *config.GithubConfig) (*Client, error) {
	githubAppKey, err := readPKCS1PrivateKey(*credentials.AppKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read github app private key: %w", err)
	}

	client := retryablehttp.NewClient()
	client.RetryMax = 3
	client.HTTPClient.Transport = ghinstallation.NewAppsTransportFromPrivateKey(
		client.HTTPClient.Transport,
		*credentials.AppID,
		githubAppKey,
	)

	return &Client{
		credentials: credentials,
		appClient:   github.NewClient(client.StandardClient()),
	}, nil
}

func (c *Client) CreateInstallationToken(
	ctx context.Context,
	installationID int64,
) (*github.InstallationToken, error) {
	ctx, span := tracer.Start(ctx, "CreateInstallationToken")
	defer span.End()

	span.SetAttributes(attribute.Int64("installation.id", installationID))

	token, _, err := c.appClient.Apps.CreateInstallationToken(ctx, installationID, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get installation token")
		return nil, fmt.Errorf("failed to get the installation token: %w", err)
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "successfully got installation token")
	return token, nil
}

func (c *Client) ParseWebhookPayload(req *http.Request) (any, error) {
	payload, err := github.ValidatePayload(req, []byte(*c.credentials.WebhookSecret))
	if err != nil {
		return nil, fmt.Errorf("invalid payload signature: %w", err)
	}

	event, err := github.ParseWebHook(github.WebHookType(req), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the github webhook payload: %w", err)
	}

	return event, nil
}

func readPKCS1PrivateKey(keyFilePath string) (*rsa.PrivateKey, error) {
	l := logger.Logger.With("keyFilePath", keyFilePath)
	l.Info("Reading Github application private key file")
	keyData, err := os.ReadFile(keyFilePath)
	if err != nil {
		return nil, err
	}

	l.Info("Decoding private key content")
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, errors.New("error decoding PEM for GithubAppKey")
	}

	l.Info("Reading private key content")
	parsedKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return parsedKey, err
}
