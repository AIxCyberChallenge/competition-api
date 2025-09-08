package otel

import (
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel/propagation"
	corev1 "k8s.io/api/core/v1"
)

// OTEL variable carrier meant to prepare for and retrieve variable values from the environment.
//
// It has 2 modus operandi:
//  1. Injection: Variables are set on it and stored internally. AsK8sVars should be used to retrieve them and set the environment on the kubernetes pod.
//  2. Extraction: Variables are retrieved from the current environment by prefix. The prefix is used to avoid collisions and identify available keys.
type EnvCarrier struct {
	vars map[string]*string
}

// Ensure `EnvCarrier` implements [propagation.TextMapCarrier]
var _ propagation.TextMapCarrier = (*EnvCarrier)(nil)

func CreateEnvCarrier() EnvCarrier {
	return EnvCarrier{vars: make(map[string]*string)}
}

const envPrefix = "ENV_CARRIER_OTEL_"

// prepend prefix and replace all - with _
func mapKey(key string) string {
	return fmt.Sprintf("%s%s", envPrefix, strings.ToUpper(strings.ReplaceAll(key, "-", "_")))
}

// strip prefix and replace all _ with - which might break if the original key contained _ intentionally
func unmapKey(mappedKey string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(mappedKey, envPrefix), "_", "-"))
}

func (c EnvCarrier) Get(key string) string {
	key = mapKey(key)
	mapVal := c.vars[key]
	if mapVal != nil {
		return *mapVal
	}

	return os.Getenv(key)
}

func (c EnvCarrier) Set(key string, value string) {
	key = mapKey(key)

	c.vars[key] = &value
}

func (c EnvCarrier) Keys() []string {
	keysSet := make(map[string]bool, len(c.vars))

	for name := range c.vars {
		keysSet[unmapKey(name)] = true
	}

	for _, env := range os.Environ() {
		split := strings.SplitN(env, "=", 2)

		if !strings.HasPrefix(split[0], envPrefix) {
			continue
		}

		keysSet[unmapKey(split[0])] = true
	}

	keys := make([]string, 0, len(keysSet))

	for k := range keysSet {
		keys = append(keys, k)
	}

	return keys
}

// Renders known variables into k8s environment var list
//
// Meant to be used after injecting the carrier vars
//
//	otel.GetTextMapPropagator().Inject(ctx, carrier)
func (c EnvCarrier) AsK8sVars() []corev1.EnvVar {
	vars := make([]corev1.EnvVar, 0, len(c.vars))

	for name, value := range c.vars {
		if value == nil {
			continue
		}

		vars = append(vars, corev1.EnvVar{Name: name, Value: *value})
	}

	return vars
}
