package templates

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BroadcastDataImages struct {
	Job string
}

type BroadcastData struct {
	Name        string
	Labels      map[string]string
	Annotations map[string]string
	Args        []string
	Images      EvaluateDataImages
	Affinity    KeyValue
	Toleration  KeyValue
	// Base64 encoded json string
	CRSAPICredentials string
	Env               []corev1.EnvVar
}

func (d BroadcastData) Render(ctx context.Context) *batchv1.Job {
	_, span := tracer.Start(ctx, "Render")
	defer span.End()

	completions := int32(1)
	ttlSecondsAfterFinished := int32(86_400)
	user := int64(1000)
	runAsNonRoot := true
	allowPrivilegeEscalation := false
	readOnlyRootFilesystem := true

	object := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        d.Name,
			Labels:      d.Labels,
			Annotations: d.Annotations,
		},
		Spec: batchv1.JobSpec{
			Completions:             &completions,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			PodFailurePolicy: &batchv1.PodFailurePolicy{
				// https://kubernetes.io/docs/tasks/job/pod-failure-policy/#using-pod-failure-policy-to-ignore-pod-disruptions
				Rules: []batchv1.PodFailurePolicyRule{
					{
						Action: batchv1.PodFailurePolicyActionFailJob,
						OnExitCodes: &batchv1.PodFailurePolicyOnExitCodesRequirement{
							Operator: batchv1.PodFailurePolicyOnExitCodesOpIn,
							Values:   []int32{1}, // 1 is unrecoverable error
						},
					},
					{
						Action: batchv1.PodFailurePolicyActionIgnore,
						OnPodConditions: []batchv1.PodFailurePolicyOnPodConditionsPattern{
							{
								Type:   corev1.DisruptionTarget,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:    &user,
						RunAsGroup:   &user,
						FSGroup:      &user,
						RunAsNonRoot: &runAsNonRoot,
					},
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: "ghcr-pull-secret",
						},
					},
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      d.Affinity.Key,
												Operator: corev1.NodeSelectorOpIn,
												Values: []string{
													d.Affinity.Value,
												},
											},
										},
									},
								},
							},
						},
					},
					Tolerations: []corev1.Toleration{
						{
							Key:      d.Toleration.Key,
							Operator: corev1.TolerationOpEqual,
							Value:    d.Toleration.Value,
							Effect:   corev1.TaintEffectNoSchedule,
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "broadcast",
							Image: d.Images.Job,
							Args:  d.Args,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
									corev1.ResourceCPU:    resource.MustParse("1"),
								},
							},
							Env: append(d.Env, corev1.EnvVar{
								Name:  "CRS_API_CREDENTIALS",
								Value: d.CRSAPICredentials,
							}),
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"ALL",
									},
								},
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
								ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "rendered broadcast job")
	return object
}
