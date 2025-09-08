package cmds

import (
	"go.opentelemetry.io/otel"
)

const name = "github.com/aixcyberchallenge/competition-api/competition-api/worker/cmds"

var tracer = otel.Tracer(name)
