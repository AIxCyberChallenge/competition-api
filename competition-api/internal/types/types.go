package types

import (
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("github.com/aixcyberchallenge/competition-api/competition-api/internal/types")

type Optional[T any] struct {
	Value   *T
	Defined bool
}

// UnmarshalJSON is implemented by deferring to the wrapped type (T).
// It will be called only if the value is defined in the JSON payload.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Defined = true
	return json.Unmarshal(data, &o.Value)
}

func (o *Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.Defined {
		return nil, nil
	}

	return json.Marshal(o.Value)
}

func Map[T any, U any](o Optional[T], f func(*T) (*U, error)) (Optional[U], error) {
	if !o.Defined {
		return Optional[U]{}, nil
	}

	v, err := f(o.Value)
	if err != nil {
		return Optional[U]{}, fmt.Errorf("failed to apply f to value: %w", err)
	}

	return Optional[U]{
		Defined: true,
		Value:   v,
	}, nil
}

func NewFromVal[T any](v T) Optional[T] {
	return Optional[T]{Defined: true, Value: &v}
}

func NewFromPtr[T any](v *T) Optional[T] {
	return Optional[T]{Defined: true, Value: v}
}
