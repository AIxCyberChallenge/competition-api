package engine

import (
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/identifier"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

type Params struct {
	resultContext    types.ResultContext
	repoDir          string
	fuzzToolingDir   string
	sanitizer        string
	architecture     string
	engine           string
	harness          string
	projectName      string
	focus            string
	allowedLanguages identifier.LanguageSlice
}

func NewParams(
	sanitizer, architecture, engine, harness, projectName, focus string,
	allowedLanguages identifier.LanguageSlice,
) Params {
	return Params{
		sanitizer:        sanitizer,
		architecture:     architecture,
		engine:           engine,
		harness:          harness,
		projectName:      projectName,
		focus:            focus,
		allowedLanguages: allowedLanguages,
	}
}

func (d Params) WithRepo(resultContext types.ResultContext, repoDir string) Params {
	d.resultContext = resultContext
	d.repoDir = repoDir

	return d
}

func (d Params) WithFuzzToolingDir(fuzzToolingDir string) Params {
	d.fuzzToolingDir = fuzzToolingDir

	return d
}
