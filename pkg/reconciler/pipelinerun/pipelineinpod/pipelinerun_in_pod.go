package pipelineinpod

import (
	"context"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/pipelinerun"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/events"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

const (
	ReasonFailedValidation   = "ReasonFailedValidation"
	ReasonCouldntGetPipeline = "CouldntGetPipeline"
)

// Reconciler implements controller.Reconciler for Run resources.
type Reconciler struct {
	pipelineClientSet clientset.Interface
	kubeClientSet     kubernetes.Interface
	pipelineRunLister listers.PipelineRunLister
}

// Check that our Reconciler implements Interface
var _ pipelinerunreconciler.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, pr *v1beta1.PipelineRun) reconciler.Event {
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling PipelineRun %s/%s at %v", pr.Namespace, pr.Name, time.Now())
	before := pr.Status.GetCondition(apis.ConditionSucceeded)

	// If the PipelineRun has not started, initialize the Condition and set the start time.
	if !pr.HasStarted() {
		logger.Infof("Starting new Run %s/%s", pr.Namespace, pr.Name)
		pr.Status.InitializeConditions()
		// In case node time was not synchronized, when controller has been scheduled to other nodes.
		if pr.Status.StartTime.Sub(pr.CreationTimestamp.Time) < 0 {
			logger.Warnf("Run %s/%s createTimestamp %s is after the PipelineRun started %s", pr.Namespace, pr.Name, pr.CreationTimestamp, pr.Status.StartTime)
			pr.Status.StartTime = &pr.CreationTimestamp
		}
		// Send the "Started" event
		afterCondition := pr.Status.GetCondition(apis.ConditionSucceeded)
		events.Emit(ctx, nil, afterCondition, pr)
	}

	getPipelineFunc, err := resources.GetPipelineFunc(ctx, r.kubeClientSet, r.pipelineClientSet, pr)
	if err != nil {
		logger.Errorf("Failed to fetch pipeline func for pipeline %s: %w", pr.Spec.PipelineRef.Name, err)
		pr.Status.MarkFailed(ReasonCouldntGetPipeline, "Error retrieving pipeline for pipelinerun %s/%s: %s",
			pr.Namespace, pr.Name, err)
		return r.finishReconcileUpdateEmitEvents(ctx, pr, before, nil)
	}

	if pr.IsDone() {
		logger.Infof("Run %s/%s is done", pr.Namespace, pr.Name)
		return r.finishReconcileUpdateEmitEvents(ctx, pr, before, nil)
	}

	// Make sure that the PipelineRun status is in sync with the actual TaskRuns
	err = r.updatePipelineRunStatusFromInformer(ctx, pr)
	if err != nil {
		// This should not fail. Return the error so we can re-try later.
		logger.Errorf("Error while syncing the pipelinerun status: %v", err.Error())
		return r.finishReconcileUpdateEmitEvents(ctx, pr, before, err)
	}

	// Reconcile this copy of the pipelinerun and then write back any status or label
	// updates regardless of whether the reconciliation errored out.
	if err = r.reconcile(ctx, pr, getPipelineFunc); err != nil {
		logger.Errorf("Reconcile error: %v", err.Error())
	}

	if err = r.finishReconcileUpdateEmitEvents(ctx, pr, before, err); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) finishReconcileUpdateEmitEvents(ctx context.Context, pr *v1beta1.PipelineRun, beforeCondition *apis.Condition, previousError error) error {
	logger := logging.FromContext(ctx)

	afterCondition := pr.Status.GetCondition(apis.ConditionSucceeded)
	events.Emit(ctx, beforeCondition, afterCondition, pr)
	_, err := r.updateLabelsAndAnnotations(ctx, pr)
	if err != nil {
		logger.Warn("Failed to update PipelineRun labels/annotations", err)
		events.EmitError(controller.GetEventRecorder(ctx), err, pr)
	}

	merr := multierror.Append(previousError, err).ErrorOrNil()
	if controller.IsPermanentError(previousError) {
		return controller.NewPermanentError(merr)
	}
	return merr
}

func (r *Reconciler) reconcile(ctx context.Context, pr *v1beta1.PipelineRun, getPipelineFunc resources.GetPipeline) error {
	logger := logging.FromContext(ctx)

	pipelineMeta, pipelineSpec, err := resources.GetPipelineData(ctx, pr, getPipelineFunc)
	if err != nil {
		logger.Errorf("Failed to determine Pipeline spec to use for pipelinerun %s: %v", pr.Name, err)
		pr.Status.MarkFailed(ReasonCouldntGetPipeline,
			"Error retrieving pipeline for pipelinerun %s/%s: %s",
			pr.Namespace, pr.Name, err)
		return controller.NewPermanentError(err)
	}

	// TODO
	// Get pod associated with pipelinerun, if it exists
	// if it does exist, update pipelinerun with status
	// if it does not exist, get pipeline from pipelinerun and use it to create a pod
	return nil
}

func validate(pr *v1beta1.PipelineRun) (errs *apis.FieldError) {
	// TODO
	return nil
}

func (c *Reconciler) updateLabelsAndAnnotations(ctx context.Context, pr *v1beta1.PipelineRun) (*v1beta1.PipelineRun, error) {
	// TODO
	return pr, nil
}
