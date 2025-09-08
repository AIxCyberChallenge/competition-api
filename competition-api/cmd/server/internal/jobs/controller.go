package jobs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/util/workqueue"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

// Encapsulates logic for the kubernetes job monitoring
type CompetitionAPIController struct {
	client    kubernetes.Interface
	db        *gorm.DB
	namespace string
	id        string
}

//nolint:ireturn // no control over the kubernetes package returning interface.
func (s *CompetitionAPIController) createInformer() (informers.SharedInformerFactory, error) {
	reqKind, err := labels.NewRequirement(
		JobKindLabel,
		selection.Equals,
		[]string{types.JobKindEval},
	)
	if err != nil {
		return nil, err
	}
	reqID, err := labels.NewRequirement(ObjectIDLabel, selection.Exists, nil)
	if err != nil {
		return nil, err
	}
	reqType, err := labels.NewRequirement(
		JobTypeLabel,
		selection.In,
		[]string{string(types.JobTypePOV), string(types.JobTypePatch), string(types.JobTypeJob)},
	)
	if err != nil {
		return nil, err
	}

	selector := labels.NewSelector().Add(*reqKind, *reqID, *reqType)

	factory := informers.NewSharedInformerFactoryWithOptions(
		s.client,
		time.Minute*15,
		informers.WithNamespace(s.namespace),
		informers.WithTweakListOptions(func(lo *metav1.ListOptions) {
			lo.LabelSelector = selector.String()
		}),
	)

	return factory, nil
}

func NewCompetitionAPIController(
	client kubernetes.Interface,
	namespace string,
	id string,
	db *gorm.DB,
) (*CompetitionAPIController, error) {
	return &CompetitionAPIController{
		namespace: namespace,
		id:        id,
		db:        db,
		client:    client,
	}, nil
}

// Runs leader election in loop until `ctx` is cancelled
func (s *CompetitionAPIController) Run(ctx context.Context) {
	lock := &resourcelock.LeaseLock{
		Client: s.client.CoordinationV1(),
		LeaseMeta: metav1.ObjectMeta{
			Name:      "competitionapi",
			Namespace: s.namespace,
		},
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: s.id,
		},
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		//nolint:govet // shadow: intentionally shadow ctx to avoid using the incorrect one.
		ctx, cancel := context.WithCancel(ctx)
		//nolint:revive // not constantly adding to defer stack in this loop
		defer cancel()
		leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
			Lock:          lock,
			LeaseDuration: time.Second * 15,
			RenewDeadline: time.Second * 10,
			RetryPeriod:   time.Second * 2,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					ctx, span := tracer.Start(
						ctx,
						"CompetitionAPIController.Run.StartedLeading",
						trace.WithNewRoot(),
					)

					logger.Logger.InfoContext(ctx, "started leading competitionapi")
					logger.Logger.InfoContext(ctx, "stating informers")

					factory, err := s.createInformer()
					if err != nil {
						cancel()
						span.RecordError(err)
						span.SetStatus(codes.Error, "failed to create informer")
						span.End()
						return
					}

					jc, err := newJobController(
						factory.Batch().V1().Jobs().Informer(),
						s.client,
						s.db,
					)
					if err != nil {
						cancel()
						span.RecordError(err)
						span.SetStatus(codes.Error, "failed to make job controller")
						span.End()
						return
					}

					factory.Start(ctx.Done())
					factory.WaitForCacheSync(ctx.Done())

					jc.run(16)

					span.RecordError(nil)
					span.SetStatus(codes.Ok, "started workers")
					span.End()

					<-ctx.Done()

					jc.shutdown()
					factory.Shutdown()
				},
				OnStoppedLeading: func() {
					logger.Logger.Info("stopped leading competitionapi")
				},
			},
		})

		<-time.After(time.Second * 30)
	}
}

// monitors jobs and processes job events
type jobController struct {
	jobInformer cache.SharedIndexInformer
	queue       workqueue.TypedRateLimitingInterface[string]
	client      kubernetes.Interface
	db          *gorm.DB
}

func newJobController(
	jobInformer cache.SharedIndexInformer,
	client kubernetes.Interface,
	db *gorm.DB,
) (*jobController, error) {
	queue := workqueue.NewTypedRateLimitingQueue(
		workqueue.DefaultTypedControllerRateLimiter[string](),
	)
	_, err := jobInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				logger.Logger.Warn("failed to make key", "obj", obj)
				return
			}

			queue.Add(key)
		},
		UpdateFunc: func(_, newObj any) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err != nil {
				logger.Logger.Warn("failed to make key", "obj", newObj)
				return
			}

			queue.Add(key)
		},
	})
	if err != nil {
		return nil, err
	}

	return &jobController{
		jobInformer: jobInformer,
		queue:       queue,
		client:      client,
		db:          db,
	}, nil
}

// spawn workers that handle job updates concurrently
func (s *jobController) run(workers int) {
	for range workers {
		go s.jobUpdateWorker(context.Background())
	}
}

func (s *jobController) shutdown() {
	s.queue.ShutDown()
}

// handles the job update event. updates db etc.
func (s *jobController) handleJobUpdate(ctx context.Context, job *batchv1.Job) error {
	ctx, span := tracer.Start(ctx, "jobController.handleJobUpdate", trace.WithAttributes(
		attribute.String("job.name", job.Name),
		attribute.String("job.namespace", job.Namespace),
	))
	defer span.End()

	failed := false
	complete := false

	for _, cond := range job.Status.Conditions {
		if (cond.Type == batchv1.JobFailed) &&
			cond.Status == corev1.ConditionTrue {
			failed = true
			// https://kubernetes.io/docs/concepts/workloads/controllers/job/#job-termination-and-cleanup
		}
		if cond.Type == batchv1.JobComplete && cond.Status == corev1.ConditionTrue {
			complete = true
		}
	}

	span.AddEvent("job_condition", trace.WithAttributes(
		attribute.Bool("failed", failed),
		attribute.Bool("complete", complete),
	))

	db := s.db.WithContext(ctx)

	if !failed && !complete {
		span.RecordError(nil)
		span.SetStatus(codes.Ok, "job not terminal")
		return nil
	}

	if failed {
		err := db.Transaction(func(db *gorm.DB) error {
			var submission models.Submission
			rawID := job.Labels[ObjectIDLabel]
			id, err := uuid.Parse(rawID)
			if err != nil {
				logger.Logger.WarnContext(ctx, "invalid object id", "error", err, "objectID", rawID)
				return nil
			}

			var result *gorm.DB
			status := types.SubmissionStatusErrored

			switch types.JobType(job.Labels[JobTypeLabel]) {
			case types.JobTypePOV:
				submission = &models.POVSubmission{}
				result = db.Model(submission).
					Clauses(clause.Returning{}).
					Where("id = ?", id).
					Where("status = ?", types.SubmissionStatusAccepted).
					Updates(
						models.POVSubmission{
							Status: status,
						},
					)
			case types.JobTypePatch:
				submission = &models.PatchSubmission{}
				result = db.Model(submission).
					Clauses(clause.Returning{}).
					Where("id = ?", id).
					Where("status = ?", types.SubmissionStatusAccepted).
					Updates(
						models.PatchSubmission{
							Status: status,
						},
					)
			case types.JobTypeJob:
				result = db.Model(&models.Job{}).
					Clauses(clause.Returning{}).
					Where("id = ?", id).
					Where("status = ?", types.SubmissionStatusAccepted).
					Updates(
						models.Job{
							Status: status,
						},
					)
			default:
				logger.Logger.WarnContext(
					ctx,
					"invalid job type not retrying",
					"type",
					job.Labels[JobTypeLabel],
				)
				return nil
			}

			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				logger.Logger.WarnContext(ctx, "event previously handled")
				return nil
			}

			if submission != nil {
				teamID := submission.GetSubmitterID().String()
				taskID := submission.GetTaskID().String()
				roundID, err := models.GetRoundIDForSubmission(ctx, db, submission)
				if err != nil {
					return err
				}
				c := audit.Context{
					TeamID:  &teamID,
					TaskID:  &taskID,
					RoundID: roundID,
				}
				submission.AuditLogSubmissionResult(c)
			}

			return nil
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to process job")
			return err
		}
	}

	propagationPolicy := metav1.DeletePropagationBackground
	err := s.client.BatchV1().
		Jobs(job.Namespace).
		Delete(ctx, job.Name, metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete job")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "cleaned up job")
	return nil
}

// pulls an update event out of the queue
func (s *jobController) processJobUpdate(ctx context.Context) bool {
	jobKey, shutdown := s.queue.Get()
	if shutdown {
		return false
	}
	defer s.queue.Done(jobKey)

	ctx, span := tracer.Start(
		ctx,
		"jobController.processJobUpdate",
		trace.WithNewRoot(),
		trace.WithLinks(trace.LinkFromContext(ctx)),
		trace.WithAttributes(attribute.String("key", jobKey)),
	)
	defer span.End()

	_, _, err := cache.SplitMetaNamespaceKey(jobKey)
	if err != nil {
		s.queue.Forget(jobKey)
		logger.Logger.WarnContext(ctx, "Invalid key format for reconciliation", "key", jobKey)
		span.RecordError(nil)
		span.SetStatus(codes.Error, "invalid key format for reconciliation")
		// return no error because we do not want to retry invalid keys
		return true
	}

	obj, exists, err := s.jobInformer.GetStore().GetByKey(jobKey)
	if err != nil {
		s.queue.Forget(jobKey)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to pull from store")
		// do not retry because failed to get job from store
		return true
	}

	if !exists {
		s.queue.Forget(jobKey)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to pull from store")
		// do not retry because job does not exist
		return true
	}

	job, ok := obj.(*batchv1.Job)
	if !ok {
		return true
	}

	err = s.handleJobUpdate(ctx, job)
	if err != nil {
		s.queue.AddRateLimited(jobKey)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to process update retrying")
		return true
	}

	s.queue.Forget(jobKey)
	span.RecordError(nil)
	span.SetStatus(codes.Ok, "successfully processed")
	return true
}

// processes job updates until the queue shutsdown
func (s *jobController) jobUpdateWorker(
	ctx context.Context,
) {
	for s.processJobUpdate(ctx) {
	}
}
