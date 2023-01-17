/*
Copyright 2022 The Tekton Authors
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

package pipelinerun

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/cancel"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

// timeoutPipelineRun marks the PipelineRun as timed out and any resolved TaskRun(s) too.
func timeoutPipelineRun(ctx context.Context, pr *v1beta1.PipelineRun, clientSet clientset.Interface) error {
	errs := cancel.CancelPipelineTaskRuns(ctx, pr, clientSet, string(v1beta1.TaskRunCancelledByPipelineTimeoutMsg))

	// If we successfully timed out all the TaskRuns and Runs, we can consider the PipelineRun timed out.
	if len(errs) == 0 {
		reason := v1beta1.PipelineRunReasonTimedOut.String()

		pr.Status.SetCondition(&apis.Condition{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionFalse,
			Reason:  reason,
			Message: fmt.Sprintf("PipelineRun %q failed to finish within %q", pr.Name, pr.PipelineTimeout(ctx).String()),
		})
		// update pr completed time
		pr.Status.CompletionTime = &metav1.Time{Time: time.Now()}
	} else {
		e := strings.Join(errs, "\n")
		// Indicate that we failed to time out the PipelineRun
		pr.Status.SetCondition(&apis.Condition{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionUnknown,
			Reason:  ReasonCouldntTimeOut,
			Message: fmt.Sprintf("PipelineRun %q was timed out but had errors trying to time out TaskRuns and/or Runs: %s", pr.Name, e),
		})
		return fmt.Errorf("error(s) from timing out TaskRun(s) from PipelineRun %s: %s", pr.Name, e)
	}
	return nil
}

func timeoutPipelineTasks(ctx context.Context, pr *v1beta1.PipelineRun, clientSet clientset.Interface) error {
	return nil
}

func timeoutFinallyTasks(ctx context.Context, pr *v1beta1.PipelineRun, clientSet clientset.Interface) error {
	return nil
}
