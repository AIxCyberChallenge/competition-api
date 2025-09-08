package types

type SourceType string

const (
	SourceTypeRepo        SourceType = "repo"
	SourceTypeFuzzTooling SourceType = "fuzz-tooling"
	SourceTypeDiff        SourceType = "diff"
)
