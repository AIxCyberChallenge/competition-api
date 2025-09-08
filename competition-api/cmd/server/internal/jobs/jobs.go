package jobs

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer(
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/jobs",
)
