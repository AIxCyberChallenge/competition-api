package jobs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/jobs/templates"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/config"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
	otelcompetitionapi "github.com/aixcyberchallenge/competition-api/competition-api/internal/otel"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

// Handles creating kubernetes jobs
type KubernetesClient struct {
	kubeClient *kubernetes.Clientset
	config     *config.Config
	namespace  string
}

const TeamLabel = "aixcc.tech/team"
const JobTypeLabel = "aixcc.tech/job-type"
const ObjectIDLabel = "aixcc.tech/object-id"
const TeamIDLabel = "aixcc.tech/team-id"
const RoundIDLabel = "aixcc.tech/round-id"
const JobKindLabel = "aixcc.tech/job-kind"

// to be used with other functions that need the job name, like Logs
func GetJobName(jobType types.JobType, thingID string) string {
	return fmt.Sprintf("%s-%s", string(jobType), thingID)
}

func (jc *KubernetesClient) createJob(
	ctx context.Context,
	job *batchv1.Job,
) (*batchv1.Job, error) {
	ctx, span := tracer.Start(ctx, "createJob")
	defer span.End()

	job, err := jc.kubeClient.BatchV1().Jobs(jc.namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create job")
		return nil, err
	}

	span.AddEvent("created_job", trace.WithAttributes(
		attribute.String("job.name", job.Name),
	))

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "created job")
	return job, nil
}

func (jc *KubernetesClient) CreateEvalJob(
	ctx context.Context,
	jobType types.JobType,
	thingID string,
	args []string,
	memoryGB int,
	cpus int,
	roundID *string,
	taskID *string,
	teamID *string,
) (*batchv1.Job, error) {
	ctx, span := tracer.Start(ctx, "CreateJob")
	defer span.End()

	annotations := map[string]string{
		"aixcc.tech/job-type": string(jobType),
	}

	name := GetJobName(jobType, thingID)
	span.SetAttributes(
		attribute.String("jobType", string(jobType)),
		attribute.String("thing.id", thingID),
		attribute.StringSlice("args", args),
		attribute.String("name", name),
	)
	l := logger.Logger.With("job_type", jobType, "name", name, "command", args)

	for k, v := range annotations {
		l = l.With(k, v)
	}

	evaluatorEnvVars := []corev1.EnvVar{
		{
			Name:  "AZURE_STORAGE_ACCOUNT_CONTAINERS_URL",
			Value: jc.config.Azure.StorageAccount.Containers.URL,
		},
		{
			Name:  "AZURE_STORAGE_ACCOUNT_NAME",
			Value: jc.config.Azure.StorageAccount.Name,
		},
		{
			Name:  "AZURE_STORAGE_ACCOUNT_KEY",
			Value: jc.config.Azure.StorageAccount.Key,
		},
		{
			Name:  "AZURE_STORAGE_ACCOUNT_CONTAINER",
			Value: jc.config.Azure.StorageAccount.Containers.Artifacts,
		},
		{
			Name:  "AZURE_STORAGE_ACCOUNT_QUEUES_URL",
			Value: jc.config.Azure.StorageAccount.Queues.URL,
		},
		{
			Name:  "AZURE_STORAGE_ACCOUNT_RESULTS_QUEUE",
			Value: jc.config.Azure.StorageAccount.Queues.Results,
		},
	}

	if roundID != nil {
		annotations[RoundIDLabel] = *roundID
		span.SetAttributes(
			attribute.String("round.id", *roundID),
		)
		evaluatorEnvVars = append(evaluatorEnvVars, corev1.EnvVar{
			Name:  "ROUND_ID",
			Value: *roundID,
		})
	}

	evaluatorEnvVars = append(evaluatorEnvVars, corev1.EnvVar{
		Name:  "GENERATE_ROUND_ID",
		Value: *jc.config.Generate.RoundID,
	})

	if taskID != nil {
		annotations["aixcc.tech/task-id"] = *taskID
		span.SetAttributes(
			attribute.String("task.id", *taskID),
		)
		evaluatorEnvVars = append(evaluatorEnvVars, corev1.EnvVar{
			Name:  "TASK_ID",
			Value: *taskID,
		})
	}

	if teamID != nil {
		annotations["aixcc.tech/team-id"] = *teamID
		span.SetAttributes(
			attribute.String("team.id", *teamID),
		)
		evaluatorEnvVars = append(evaluatorEnvVars, corev1.EnvVar{
			Name:  "TEAM_ID",
			Value: *teamID,
		})
	}

	if jobType == types.JobTypePOV {
		evaluatorEnvVars = append(evaluatorEnvVars, corev1.EnvVar{
			Name:  "POV_ID",
			Value: thingID,
		})

		annotations["aixcc.tech/pov-id"] = thingID
	}

	if jobType == types.JobTypePatch {
		evaluatorEnvVars = append(evaluatorEnvVars, corev1.EnvVar{
			Name:  "PATCH_ID",
			Value: thingID,
		})

		annotations["aixcc.tech/patch-id"] = thingID
	}

	labels := map[string]string{
		JobTypeLabel:  string(jobType),
		ObjectIDLabel: thingID,
		JobKindLabel:  types.JobKindEval,
	}

	carrier := otelcompetitionapi.CreateEnvCarrier()
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	evaluatorEnvVars = append(evaluatorEnvVars, carrier.AsK8sVars()...)
	evaluatorEnvVars = append(evaluatorEnvVars,
		corev1.EnvVar{
			Name:  "USE_OTLP",
			Value: strconv.FormatBool(jc.config.Logging.UseOTLP),
		},
		corev1.EnvVar{
			Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
			Value: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		},
		corev1.EnvVar{
			Name:  "OTEL_RESOURCE_ATTRIBUTES",
			Value: os.Getenv("OTEL_RESOURCE_ATTRIBUTES"),
		},
		corev1.EnvVar{
			Name:  "OTEL_SERVICE_NAME",
			Value: "competitionapi-worker",
		},
	)

	nodeAssignment := jc.config.K8s.EvalNodeAssignment

	// scoring job
	if jobType == types.JobTypeJob {
		nodeAssignment = jc.config.K8s.ScoringNodeAssignment
	}

	data := templates.EvaluateData{
		Name:        name,
		Labels:      labels,
		Annotations: annotations,
		Args:        args,
		Env:         evaluatorEnvVars,
		Images: templates.EvaluateDataImages{
			Job:  jc.config.K8s.JobImage,
			DIND: jc.config.K8s.DINDImage,
		},
		TeamID:      teamID,
		TeamIDLabel: TeamIDLabel,
		Affinity: templates.KeyValue{
			Key:   nodeAssignment.NodeAffinityLabel.Key,
			Value: nodeAssignment.NodeAffinityLabel.Value,
		},
		Toleration: templates.KeyValue{
			Key:   nodeAssignment.Toleration.Key,
			Value: nodeAssignment.Toleration.Value,
		},
		DindMemoryGB: memoryGB,
		DindCPUs:     cpus,
	}

	job, err := jc.createJob(ctx, data.Render(ctx))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create eval job")
		return nil, err
	}

	span.AddEvent("created_eval_job")

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "created job")
	return job, nil
}

func (jc *KubernetesClient) CreateDeliveryJob(
	ctx context.Context,
	route string,
	crsCredentials []config.Team,
	roundID string,
	taskID string,
	messageID string,
	deadline time.Time,
	payload []byte,
) (*batchv1.Job, error) {
	ctx, span := tracer.Start(ctx, "CreateDeliveryJob")
	span.End()

	teamIDs := make([]string, 0, len(crsCredentials))
	for _, crs := range crsCredentials {
		teamIDs = append(teamIDs, crs.ID)
	}

	span.SetAttributes(
		attribute.String("route", route),
		attribute.String("round.id", roundID),
		attribute.String("task.id", taskID),
		attribute.String("message.id", messageID),
		attribute.Int64("deadline", deadline.UnixMilli()),
		attribute.String("payload", string(payload)),
		attribute.StringSlice("teams", teamIDs),
	)

	crsAPIDetails, err := json.Marshal(crsCredentials)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to serialize crs details")
		return nil, errors.New("failed to serialize crs details")
	}

	annotations := map[string]string{
		"aixcc.tech/round-id": roundID,
		"aixcc.tech/task-id":  taskID,
	}

	labels := map[string]string{
		JobKindLabel: types.JobKindBroadcast,
	}

	args := []string{
		"worker",
		"broadcast",
		"-r",
		route,
		"-p",
		base64.StdEncoding.EncodeToString(payload),
		"-d",
		roundID,
		"-t",
		taskID,
		"-u",
		strconv.FormatInt(deadline.Unix(), 10),
	}

	envVars := []corev1.EnvVar{}

	carrier := otelcompetitionapi.CreateEnvCarrier()
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	envVars = append(envVars, carrier.AsK8sVars()...)
	envVars = append(envVars,
		corev1.EnvVar{
			Name:  "USE_OTLP",
			Value: strconv.FormatBool(jc.config.Logging.UseOTLP),
		},
		corev1.EnvVar{
			Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
			Value: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		},
		corev1.EnvVar{
			Name:  "OTEL_RESOURCE_ATTRIBUTES",
			Value: os.Getenv("OTEL_RESOURCE_ATTRIBUTES"),
		},
		corev1.EnvVar{
			Name:  "OTEL_SERVICE_NAME",
			Value: "competitionapi-worker",
		},
	)

	crsAPICredentials := base64.StdEncoding.EncodeToString(crsAPIDetails)

	data := templates.BroadcastData{
		Name:        fmt.Sprintf("broadcast-%s", messageID),
		Labels:      labels,
		Annotations: annotations,
		Args:        args,
		Env:         envVars,
		Images: templates.EvaluateDataImages{
			Job: jc.config.K8s.JobImage,
		},
		Affinity: templates.KeyValue{
			Key:   jc.config.K8s.BroadcastNodeAssignment.NodeAffinityLabel.Key,
			Value: jc.config.K8s.BroadcastNodeAssignment.NodeAffinityLabel.Value,
		},
		Toleration: templates.KeyValue{
			Key:   jc.config.K8s.BroadcastNodeAssignment.Toleration.Key,
			Value: jc.config.K8s.BroadcastNodeAssignment.Toleration.Value,
		},
		CRSAPICredentials: crsAPICredentials,
	}

	job, err := jc.createJob(ctx, data.Render(ctx))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create job")
		return nil, err
	}

	span.AddEvent("created_delivery_job")

	return job, nil
}

func (jc *KubernetesClient) CreateCancelJob(
	ctx context.Context,
	route string,
	crsCredentials []config.Team,
	roundID string,
	deadline time.Time,
) (*batchv1.Job, error) {
	ctx, span := tracer.Start(ctx, "CreateCancelJob")
	defer span.End()

	teamIDs := make([]string, 0, len(crsCredentials))
	for _, crs := range crsCredentials {
		teamIDs = append(teamIDs, crs.ID)
	}

	span.SetAttributes(
		attribute.String("route", route),
		attribute.String("round.id", roundID),
		attribute.Int64("deadline", deadline.UnixMilli()),
		attribute.StringSlice("teams", teamIDs),
	)

	crsAPIDetails, err := json.Marshal(crsCredentials)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to serialize crs details")
		return nil, errors.New("failed to serialize crs details")
	}

	annotations := map[string]string{
		"aixcc.tech/round-id": roundID,
	}

	labels := map[string]string{
		JobKindLabel: types.JobKindCancel,
	}

	args := []string{
		"worker",
		"cancel",
		"-r",
		route,
		"-d",
		roundID,
		"-u",
		strconv.FormatInt(deadline.Unix(), 10),
	}

	envVars := []corev1.EnvVar{}

	carrier := otelcompetitionapi.CreateEnvCarrier()
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	envVars = append(envVars, carrier.AsK8sVars()...)
	envVars = append(envVars,
		corev1.EnvVar{
			Name:  "USE_OTLP",
			Value: strconv.FormatBool(jc.config.Logging.UseOTLP),
		},
		corev1.EnvVar{
			Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
			Value: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		},
		corev1.EnvVar{
			Name:  "OTEL_RESOURCE_ATTRIBUTES",
			Value: os.Getenv("OTEL_RESOURCE_ATTRIBUTES"),
		},
		corev1.EnvVar{
			Name:  "OTEL_SERVICE_NAME",
			Value: "competitionapi-worker",
		},
	)

	crsAPICredentials := base64.StdEncoding.EncodeToString(crsAPIDetails)

	data := templates.BroadcastData{
		Name:        fmt.Sprintf("cancel-%s", uuid.New().String()),
		Labels:      labels,
		Annotations: annotations,
		Args:        args,
		Env:         envVars,
		Images: templates.EvaluateDataImages{
			Job: jc.config.K8s.JobImage,
		},
		Affinity: templates.KeyValue{
			Key:   jc.config.K8s.BroadcastNodeAssignment.NodeAffinityLabel.Key,
			Value: jc.config.K8s.BroadcastNodeAssignment.NodeAffinityLabel.Value,
		},
		Toleration: templates.KeyValue{
			Key:   jc.config.K8s.BroadcastNodeAssignment.Toleration.Key,
			Value: jc.config.K8s.BroadcastNodeAssignment.Toleration.Value,
		},
		CRSAPICredentials: crsAPICredentials,
	}

	job, err := jc.createJob(ctx, data.Render(ctx))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create cancel job")
		return nil, err
	}

	span.AddEvent("created_cancel_job")

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "created cancel job")
	return job, err
}

func CreateJobClient(
	namespace string,
	client *kubernetes.Clientset,
	cfg *config.Config,
) KubernetesClient {
	return KubernetesClient{
		namespace:  namespace,
		kubeClient: client,
		config:     cfg,
	}
}
