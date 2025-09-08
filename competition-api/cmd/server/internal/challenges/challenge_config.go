package challenges

import (
	"time"

	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

type ChallengeConfig struct {
	BaseRef      *string
	AuthMethod   githttp.BasicAuth
	Name         string
	RepoURL      string
	HeadRef      string
	TaskDuration time.Duration
}
