package templates

import (
	"go.opentelemetry.io/otel"
)

const tracerName = "github.com/aixcyberchallenge/competition-api/competition-api/jobs/templates"

var tracer = otel.Tracer(tracerName)

type KeyValue struct {
	Key   string
	Value string
}
