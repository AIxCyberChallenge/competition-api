package types

import (
	"context"
	"os"
	"path/filepath"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v2"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/validator"
)

type ChallengeYAML struct {
	FuzzToolingProjectName string   `yaml:"fuzz_tooling_project_name" validate:"required"`
	FuzzToolingURL         string   `yaml:"fuzz_tooling_url"          validate:"required"`
	FuzzToolingRef         string   `yaml:"fuzz_tooling_ref"          validate:"required"`
	HarnessesList          []string `yaml:"harnesses"`
	MemoryGB               int      `yaml:"required_memory_gb"`
	CPUs                   int      `yaml:"cpus"`
}

func ParseChallengeYAML(
	ctx context.Context,
	repoPath string,
) (*ChallengeYAML, error) {
	_, span := tracer.Start(ctx, "parseChallengeYAML", trace.WithAttributes(
		attribute.String("repo.path", repoPath),
	))
	defer span.End()

	challYamlPath := filepath.Join(repoPath, ".aixcc", "challenge.yaml")
	span.SetAttributes(attribute.String("repo.challengeYaml.Path", challYamlPath))

	var content []byte
	content, err := os.ReadFile(challYamlPath)
	if err != nil {
		span.SetStatus(codes.Error, "error reading file")
		span.RecordError(err)
		return nil, err
	}

	cYAMLdata := ChallengeYAML{
		MemoryGB: 16,
		CPUs:     2,
	}
	err = yaml.Unmarshal(content, &cYAMLdata)
	if err != nil {
		span.SetStatus(codes.Error, "error unmarshalling challenge yaml")
		span.RecordError(err)
		return nil, err
	}

	span.AddEvent("validating parsed challenge YAML")
	v := validator.Create()
	err = v.Validate(cYAMLdata)
	if err != nil {
		span.SetStatus(codes.Error, "error validating challenge yaml")
		span.RecordError(err)
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	span.RecordError(nil)
	return &cYAMLdata, nil
}
