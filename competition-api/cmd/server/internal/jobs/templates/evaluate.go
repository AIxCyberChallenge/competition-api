package templates

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/codes"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EvaluateDataImages struct {
	Job  string
	DIND string
}

type EvaluateData struct {
	Labels       map[string]string
	Annotations  map[string]string
	TeamID       *string
	Images       EvaluateDataImages
	Affinity     KeyValue
	Toleration   KeyValue
	Name         string
	TeamIDLabel  string
	Args         []string
	Env          []corev1.EnvVar
	DindMemoryGB int
	DindCPUs     int
}

func (d EvaluateData) Render(ctx context.Context) *batchv1.Job {
	_, span := tracer.Start(ctx, "Render")
	defer span.End()

	completions := int32(1)
	backoff := int32(0)
	shareProcessNamespace := true
	privileged := true
	user := int64(1000)
	runAsNonRoot := true
	allowPrivilegeEscalation := false
	always := corev1.ContainerRestartPolicyAlways

	nodeAffinity := corev1.NodeAffinity{
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
	}

	tolerations := []corev1.Toleration{
		{
			Key:      d.Toleration.Key,
			Operator: corev1.TolerationOpEqual,
			Value:    d.Toleration.Value,
			Effect:   corev1.TaintEffectNoSchedule,
		},
	}

	if d.TeamID != nil {
		tolerations = append(tolerations, corev1.Toleration{
			Key:      d.TeamIDLabel,
			Operator: corev1.TolerationOpEqual,
			Value:    *d.TeamID,
			Effect:   corev1.TaintEffectNoSchedule,
		})

		nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = []corev1.PreferredSchedulingTerm{
			{
				Weight: 10,
				Preference: corev1.NodeSelectorTerm{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      d.TeamIDLabel,
							Operator: corev1.NodeSelectorOpIn,
							Values: []string{
								*d.TeamID,
							},
						},
					},
				},
			},
		}
	}

	var object = &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        d.Name,
			Labels:      d.Labels,
			Annotations: d.Annotations,
		},
		Spec: batchv1.JobSpec{
			Completions:  &completions,
			BackoffLimit: &backoff,
			PodFailurePolicy: &batchv1.PodFailurePolicy{
				// https://kubernetes.io/docs/tasks/job/pod-failure-policy/#using-pod-failure-policy-to-ignore-pod-disruptions
				Rules: []batchv1.PodFailurePolicyRule{
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
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: "ghcr-pull-secret",
						},
					},
					ShareProcessNamespace: &shareProcessNamespace,
					Affinity: &corev1.Affinity{
						NodeAffinity: &nodeAffinity,
					},
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: &user,
					},
					Tolerations: tolerations,
					InitContainers: []corev1.Container{
						{
							RestartPolicy: &always,
							Name:          "dind",
							Image:         d.Images.DIND,
							Env: []corev1.EnvVar{
								{
									Name:  "DOCKER_TLS_CERTDIR",
									Value: "",
								},
								{
									Name:  "DOCKER_HOST",
									Value: "tcp://127.0.0.1:2375",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "dind-shared",
									MountPath: "/dind-shared",
								},
								{
									Name:      "dind-data",
									MountPath: "/var/lib/docker",
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(
										fmt.Sprintf("%dGi", d.DindMemoryGB),
									),
									corev1.ResourceCPU: resource.MustParse(
										fmt.Sprintf("%d", d.DindCPUs),
									),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(
										fmt.Sprintf("%dGi", d.DindMemoryGB),
									),
								},
							},
							StartupProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"docker", "info"},
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"docker", "info"},
									},
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"docker", "info"},
									},
								},
							},
						},
						{
							Name:  "load-dind",
							Image: d.Images.DIND,
							Command: []string{
								"docker",
								"load",
								"-i",
								"/app/images.tar.gz",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "DOCKER_TLS_CERTDIR",
									Value: "",
								},
								{
									Name:  "DOCKER_HOST",
									Value: "tcp://127.0.0.1:2375",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "dind-shared",
									MountPath: "/dind-shared",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
									corev1.ResourceCPU:    resource.MustParse("250m"),
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "evaluator",
							Image: d.Images.Job,
							Args: append(
								[]string{"worker", "eval", "--base-dir", "/dind-shared"},
								d.Args...),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "dind-shared",
									MountPath: "/dind-shared",
								},
							},
							Env: append(d.Env, corev1.EnvVar{
								Name:  "DOCKER_HOST",
								Value: "tcp://127.0.0.1:2375",
							}),
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:                &user,
								RunAsGroup:               &user,
								RunAsNonRoot:             &runAsNonRoot,
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"ALL",
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
									corev1.ResourceCPU:    resource.MustParse("500m"),
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes: []corev1.Volume{
						{
							Name: "dind-shared",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "dind-data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "rendered evaluate job")
	return object
}
