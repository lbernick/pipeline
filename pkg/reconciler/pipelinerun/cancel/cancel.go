/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cancel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/pipelinerunmetrics"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
)

const (
	// ReasonCouldntCancel indicates that a PipelineRun was cancelled but attempting to update
	// all of the running TaskRuns as cancelled failed.
	ReasonCouldntCancel = "PipelineRunCouldntCancel"
	// ReasonCancelled indicates that a PipelineRun was cancelled.
	ReasonCancelled = pipelinerunmetrics.ReasonCancelled
)

func cancelCustomRun(ctx context.Context, runName string, namespace string, clientSet clientset.Interface, patchBytes []byte) error {
	_, err := clientSet.TektonV1beta1().CustomRuns(namespace).Patch(ctx, runName, types.JSONPatchType, patchBytes, metav1.PatchOptions{}, "")
	if errors.IsNotFound(err) {
		// The resource may have been deleted in the meanwhile, but we should
		// still be able to cancel the PipelineRun
		return nil
	}
	return err
}

func cancelRun(ctx context.Context, runName string, namespace string, clientSet clientset.Interface, patchBytes []byte) error {
	_, err := clientSet.TektonV1alpha1().Runs(namespace).Patch(ctx, runName, types.JSONPatchType, patchBytes, metav1.PatchOptions{}, "")
	if errors.IsNotFound(err) {
		// The resource may have been deleted in the meanwhile, but we should
		// still be able to cancel the PipelineRun
		return nil
	}
	return err
}

func cancelTaskRun(ctx context.Context, taskRunName string, namespace string, clientSet clientset.Interface, patchBytes []byte) error {
	_, err := clientSet.TektonV1beta1().TaskRuns(namespace).Patch(ctx, taskRunName, types.JSONPatchType, patchBytes, metav1.PatchOptions{}, "")
	if errors.IsNotFound(err) {
		// The resource may have been deleted in the meanwhile, but we should
		// still be able to cancel the PipelineRun
		return nil
	}
	return err
}

// CancelAndFinishPipelineRun marks the PipelineRun as cancelled and any resolved TaskRun(s) too.
// TODO better docstring
func CancelAndFinishPipelineRun(ctx context.Context, pr *v1beta1.PipelineRun, clientSet clientset.Interface, reason string) error {
	errs := CancelPipelineTaskRuns(ctx, pr, clientSet, reason)

	// If we successfully cancelled all the TaskRuns and Runs, we can consider the PipelineRun cancelled.
	if len(errs) == 0 {
		reason := ReasonCancelled

		pr.Status.SetCondition(&apis.Condition{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionFalse,
			Reason:  reason,
			Message: fmt.Sprintf("PipelineRun %q was cancelled", pr.Name),
		})
		// update pr completed time
		pr.Status.CompletionTime = &metav1.Time{Time: time.Now()}
	} else {
		e := strings.Join(errs, "\n")
		// Indicate that we failed to cancel the PipelineRun
		pr.Status.SetCondition(&apis.Condition{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionUnknown,
			Reason:  ReasonCouldntCancel,
			Message: fmt.Sprintf("PipelineRun %q was cancelled but had errors trying to cancel TaskRuns and/or Runs: %s", pr.Name, e),
		})
		return fmt.Errorf("error(s) from cancelling TaskRun(s) from PipelineRun %s: %s", pr.Name, e)
	}
	return nil
}

// CancelPipelineTaskRuns patches `TaskRun` and `Run` with canceled status
// TODO: docstring
// TODO: get rid of set of tasknames
func CancelPipelineTaskRuns(ctx context.Context, pr *v1beta1.PipelineRun, clientSet clientset.Interface, reason string) []string {
	return cancelPipelineTaskRunsForTaskNames(ctx, pr, clientSet, sets.NewString(), reason)
}

// TODO: remove as a separate function
// cancelPipelineTaskRunsForTaskNames patches `TaskRun`s and `Run`s for the given task names, or all if no task names are given, with canceled status
func cancelPipelineTaskRunsForTaskNames(ctx context.Context, pr *v1beta1.PipelineRun, clientSet clientset.Interface, taskNames sets.String, reason string) []string {
	errs := []string{}
	logger := logging.FromContext(ctx)
	patchBytes, err := getPatchBytes(ctx, reason)
	if err != nil {
		return []string{err.Error()}
	}

	trNames, customRunNames, runNames, err := getChildObjectsFromPRStatusForTaskNames(ctx, pr.Status, taskNames)
	if err != nil {
		errs = append(errs, err.Error())
	}

	for _, taskRunName := range trNames {
		logger.Infof("cancelling TaskRun %s", taskRunName)

		if err := cancelTaskRun(ctx, taskRunName, pr.Namespace, clientSet, patchBytes); err != nil {
			errs = append(errs, fmt.Errorf("failed to patch TaskRun `%s` with cancellation: %s", taskRunName, err).Error())
			continue
		}
	}

	for _, runName := range customRunNames {
		logger.Infof("cancelling CustomRun %s", runName)

		if err := cancelCustomRun(ctx, runName, pr.Namespace, clientSet, patchBytes); err != nil {
			errs = append(errs, fmt.Errorf("failed to patch CustomRun `%s` with cancellation: %s", runName, err).Error())
			continue
		}
	}

	for _, runName := range runNames {
		logger.Infof("cancelling Run %s", runName)

		if err := cancelRun(ctx, runName, pr.Namespace, clientSet, patchBytes); err != nil {
			errs = append(errs, fmt.Errorf("failed to patch Run `%s` with cancellation: %s", runName, err).Error())
			continue
		}
	}

	return errs
}

// getChildObjectsFromPRStatusForTaskNames returns taskruns, customruns, and runs in the PipelineRunStatus's ChildReferences or TaskRuns/Runs,
// based on the value of the embedded status flag and the given set of PipelineTask names. If that set is empty, all are returned.
func getChildObjectsFromPRStatusForTaskNames(ctx context.Context, prs v1beta1.PipelineRunStatus, taskNames sets.String) ([]string, []string, []string, error) {
	cfg := config.FromContextOrDefaults(ctx)

	var trNames []string
	var customRunNames []string
	var runNames []string
	unknownChildKinds := make(map[string]string)

	if cfg.FeatureFlags.EmbeddedStatus != config.FullEmbeddedStatus {
		for _, cr := range prs.ChildReferences {
			if taskNames.Len() == 0 || taskNames.Has(cr.PipelineTaskName) {
				switch cr.Kind {
				case pipeline.TaskRunControllerName:
					trNames = append(trNames, cr.Name)
				case pipeline.RunControllerName:
					runNames = append(runNames, cr.Name)
				case pipeline.CustomRunControllerName:
					customRunNames = append(customRunNames, cr.Name)
				default:
					unknownChildKinds[cr.Name] = cr.Kind
				}
			}
		}
	} else {
		for trName, trs := range prs.TaskRuns {
			if taskNames.Len() == 0 || taskNames.Has(trs.PipelineTaskName) {
				trNames = append(trNames, trName)
			}
		}
		for runName, runStatus := range prs.Runs {
			if taskNames.Len() == 0 || taskNames.Has(runStatus.PipelineTaskName) {
				if cfg.FeatureFlags.CustomTaskVersion == config.CustomTaskVersionAlpha {
					runNames = append(runNames, runName)
				} else {
					customRunNames = append(customRunNames, runName)
				}
			}
		}
	}

	var err error
	if len(unknownChildKinds) > 0 {
		err = fmt.Errorf("found child objects of unknown kinds: %v", unknownChildKinds)
	}

	return trNames, customRunNames, runNames, err
}

// CancelPipelineRun marks any non-final resolved TaskRun(s) as cancelled and runs finally.
// TODO better docstring
func CancelPipelineRun(ctx context.Context, pr *v1beta1.PipelineRun, clientSet clientset.Interface, reason string) error {
	errs := CancelPipelineTaskRuns(ctx, pr, clientSet, reason)

	// If we successfully cancelled all the TaskRuns and Runs, we can proceed with the PipelineRun reconciliation to trigger finally.
	if len(errs) > 0 {
		e := strings.Join(errs, "\n")
		// Indicate that we failed to cancel the PipelineRun
		pr.Status.SetCondition(&apis.Condition{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionUnknown,
			Reason:  ReasonCouldntCancel,
			Message: fmt.Sprintf("PipelineRun %q was cancelled but had errors trying to cancel TaskRuns and/or Runs: %s", pr.Name, e),
		})
		return fmt.Errorf("error(s) from cancelling TaskRun(s) from PipelineRun %s: %s", pr.Name, e)
	}
	return nil
}

func getPatchBytes(ctx context.Context, reason string) ([]byte, error) {
	// Because the reason is a string, JSON marshalling should not fail,
	// and the controller should not panic.
	return json.Marshal([]jsonpatch.JsonPatchOperation{
		{
			Operation: "add",
			Path:      "/spec/status",
			Value:     v1beta1.CustomRunSpecStatusCancelled,
		},
		{
			Operation: "add",
			Path:      "/spec/statusMessage",
			Value:     reason,
		}})
}
