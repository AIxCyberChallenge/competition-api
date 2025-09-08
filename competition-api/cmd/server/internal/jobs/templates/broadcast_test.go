package templates

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBroadcast(t *testing.T) {
	t.Run("Render", func(t *testing.T) {
		data := BroadcastData{
			Name: "foobar-name",
			Labels: map[string]string{
				"testing.com/label": "success",
			},
			Annotations: map[string]string{
				"testing.com/annotation": "success",
			},
			Args: []string{"echo", "success", "\nfoo\"bar:\\"},
			Images: EvaluateDataImages{
				Job:  "jobImage",
				DIND: "jobImage",
			},
			Affinity: KeyValue{
				Key:   "afinity-key",
				Value: "affinity-value",
			},
			Toleration: KeyValue{
				Key:   "toleration-key",
				Value: "toleration-value",
			},
			CRSAPICredentials: base64.StdEncoding.EncodeToString([]byte("i am very credentialed")),
		}

		jobSpec := data.Render(context.TODO())

		assert.Equal(
			t,
			"toleration-value",
			jobSpec.Spec.Template.Spec.Tolerations[0].Value,
			"toleration is not set",
		)
		assert.Equal(
			t,
			"affinity-value",
			jobSpec.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0],
			"affinity is not set",
		)
		assert.Equal(
			t,
			"\nfoo\"bar:\\",
			jobSpec.Spec.Template.Spec.Containers[0].Args[2],
			"arg value was not escaped",
		)
	})
}
