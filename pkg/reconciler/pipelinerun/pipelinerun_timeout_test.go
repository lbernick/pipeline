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
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test"
	"github.com/tektoncd/pipeline/test/diff"
	"github.com/tektoncd/pipeline/test/names"
	"github.com/tektoncd/pipeline/test/parse"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktesting "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestReconcileForCustomTaskWithPipelineTaskTimedOut(t *testing.T) {
	names.TestingSeed()
	// TestReconcileForCustomTaskWithPipelineTaskTimedOut runs "Reconcile" on a PipelineRun.
	// It verifies that reconcile is successful, and the individual
	// custom task which has timed out, is patched as cancelled.
	ps := []*v1beta1.Pipeline{parse.MustParsePipeline(t, `
metadata:
  name: test-pipeline
  namespace: test
spec:
  tasks:
  - name: hello-world-1
    taskRef:
      apiVersion: example.dev/v0
      kind: Example
`)}
	prName := "test-pipeline-run-custom-task-with-timeout"
	prs := []*v1beta1.PipelineRun{parse.MustParsePipelineRun(t, `
metadata:
  name: test-pipeline-run-custom-task-with-timeout
  namespace: test
spec:
  pipelineRef:
    name: test-pipeline
  serviceAccountName: test-sa
`)}
	runs := []*v1alpha1.Run{mustParseRunWithObjectMeta(t,
		taskRunObjectMeta("test-pipeline-run-custom-task-with-timeout-hello-world-1", "test", "test-pipeline-run-custom-task-with-timeout",
			"test-pipeline", "hello-world-1", true),
		`
spec:
  ref:
    apiVersion: example.dev/v0
    kind: Example
  timeout: 1m0s
status:
  conditions:
  - status: Unknown
    type: Succeeded
  startTime: "2021-12-31T23:58:59Z"
`)}
	cms := []*corev1.ConfigMap{withCustomTasks(newFeatureFlagsConfigMap())}
	d := test.Data{
		PipelineRuns: prs,
		Pipelines:    ps,
		ConfigMaps:   cms,
		Runs:         runs,
	}
	prt := newPipelineRunTest(d, t)
	defer prt.Cancel()

	wantEvents := []string{
		"Normal Started",
		"Normal Running Tasks Completed: 0 \\(Failed: 0, Cancelled 0\\), Incomplete: 1, Skipped: 0",
	}
	_, clients := prt.reconcileRun("test", prName, wantEvents, false)

	actions := clients.Pipeline.Actions()
	if len(actions) < 2 {
		t.Fatalf("Expected client to have at least two action implementation but it has %d", len(actions))
	}

	// The patch operation to cancel the run must be executed.
	var got []jsonpatch.Operation
	for _, a := range actions {
		if action, ok := a.(ktesting.PatchAction); ok {
			if a.(ktesting.PatchAction).Matches("patch", "runs") {
				err := json.Unmarshal(action.GetPatch(), &got)
				if err != nil {
					t.Fatalf("Expected to get a patch operation for cancel,"+
						" but got error: %v\n", err)
				}
				break
			}
		}
	}
	want := []jsonpatch.JsonPatchOperation{{
		Operation: "add",
		Path:      "/spec/status",
		Value:     "RunCancelled",
	}}
	if d := cmp.Diff(got, want); d != "" {
		t.Fatalf("Expected cancel patch operation, but got a mismatch %s", diff.PrintWantGot(d))
	}
}

// TODO: the following test but also for finally tasks
func TestReconcileForCustomTaskWithPipelineRunTimedOut(t *testing.T) {
	names.TestingSeed()
	// TestReconcileForCustomTaskWithPipelineRunTimedOut runs "Reconcile" on a
	// PipelineRun that has timed out.
	// It verifies that reconcile is successful, and the custom task has also timed
	// out and patched as cancelled.
	for _, tc := range []struct {
		name           string
		timeout        *metav1.Duration
		timeouts       *v1beta1.TimeoutFields
		retries        int
		runStatus      *v1alpha1.RunStatus
		wantRunTimeout time.Duration
	}{{
		name:           "spec.Timeout",
		timeout:        &metav1.Duration{Duration: 12 * time.Hour},
		wantRunTimeout: time.Second,
	}, {
		name:           "spec.Timeouts.Pipeline; run not started",
		timeouts:       &v1beta1.TimeoutFields{Pipeline: &metav1.Duration{Duration: 12 * time.Hour}},
		wantRunTimeout: time.Second,
	}, {
		name:     "spec.Timeouts.Pipeline; run already created",
		timeouts: &v1beta1.TimeoutFields{Pipeline: &metav1.Duration{Duration: 12 * time.Hour}},
		runStatus: &v1alpha1.RunStatus{Status: duckv1.Status{
			Conditions: []apis.Condition{{Type: apis.ConditionSucceeded, Status: corev1.ConditionUnknown}}},
		},
		wantRunTimeout: 3 * time.Hour,
	}, {
		name:     "spec.Timeouts.Pipeline; run already finished",
		timeouts: &v1beta1.TimeoutFields{Pipeline: &metav1.Duration{Duration: 12 * time.Hour}},
		runStatus: &v1alpha1.RunStatus{Status: duckv1.Status{
			Conditions: []apis.Condition{{Type: apis.ConditionSucceeded, Status: corev1.ConditionTrue}}},
		},
		wantRunTimeout: 3 * time.Hour,
	}, {
		name:     "spec.Timeouts.Pipeline; run already finished some of its attempts",
		timeouts: &v1beta1.TimeoutFields{Pipeline: &metav1.Duration{Duration: 12 * time.Hour}},
		retries:  2,
		runStatus: &v1alpha1.RunStatus{Status: duckv1.Status{
			Conditions: []apis.Condition{{Type: apis.ConditionSucceeded, Status: corev1.ConditionFalse}}},
			RunStatusFields: v1alpha1.RunStatusFields{
				RetriesStatus: []v1alpha1.RunStatus{{}},
			},
		},
		wantRunTimeout: 3 * time.Hour,
	}, {
		name:     "spec.Timeouts.Pipeline; run created with wrong timeout",
		timeouts: &v1beta1.TimeoutFields{Pipeline: &metav1.Duration{Duration: 23 * time.Hour}},
		runStatus: &v1alpha1.RunStatus{Status: duckv1.Status{
			Conditions: []apis.Condition{{Type: apis.ConditionSucceeded, Status: corev1.ConditionFalse}}},
			RunStatusFields: v1alpha1.RunStatusFields{
				// PipelineRun started 1 day ago with a timeout of 23 hours
				// Run started 2 hours ago, so it should have a timeout of 1 hour,
				// but instead it has a timeout of 3 hours
				// This should not happen but it does (#4071)
				StartTime: &metav1.Time{Time: now.Add(-2 * time.Hour)},
			},
		},
		wantRunTimeout: 1 * time.Second,
	}, /*{
		name:           "spec.Timeouts.Tasks",
		timeouts:       &v1beta1.TimeoutFields{Tasks: &metav1.Duration{Duration: 12 * time.Hour}},
		wantRunTimeout: 5 * time.Minute,
	} */} {
		t.Run(tc.name, func(*testing.T) {
			ps := []*v1beta1.Pipeline{parse.MustParsePipeline(t, fmt.Sprintf(`
metadata:
  name: test-pipeline
  namespace: test
spec:
  tasks:
  - name: hello-world-1
    taskRef:
      apiVersion: example.dev/v0
      kind: Example
    retries: %d
`, tc.retries))}

			prName := "test-pipeline-run-custom-task"
			prs := []*v1beta1.PipelineRun{parse.MustParsePipelineRun(t, `
metadata:
  name: test-pipeline-run-custom-task
  namespace: test
spec:
  pipelineRef:
    name: test-pipeline
  serviceAccountName: test-sa
status:
  startTime: "2021-12-31T00:00:00Z"
`)}
			prs[0].Spec.Timeout = tc.timeout
			prs[0].Spec.Timeouts = tc.timeouts

			cms := []*corev1.ConfigMap{withCustomTasks(newFeatureFlagsConfigMap())}
			run := parse.MustParseRun(t, `
metadata:
  name: test-pipeline-run-custom-task-hello-world-1
  namespace: test
spec:
  ref:
    apiVersion: example.dev/v0
    kind: Example
  timeout: 3h
`)
			var runs []*v1alpha1.Run
			if tc.runStatus != nil {
				run.Status = *tc.runStatus
				runs = []*v1alpha1.Run{run}
				prs[0].Status.ChildReferences = []v1beta1.ChildStatusReference{{Name: run.Name}}
				prs[0].Status.Runs = map[string]*v1beta1.PipelineRunRunStatus{run.Name: {PipelineTaskName: "hello-world-1"}}
			}
			d := test.Data{
				PipelineRuns: prs,
				Pipelines:    ps,
				ConfigMaps:   cms,
				Runs:         runs,
			}
			prt := newPipelineRunTest(d, t)
			defer prt.Cancel()

			wantEvents := []string{
				fmt.Sprintf("Warning Failed PipelineRun \"%s\" failed to finish within \"12h0m0s\"", prName),
			}
			runName := "test-pipeline-run-custom-task-hello-world-1"

			reconciledRun, clients := prt.reconcileRun("test", prName, wantEvents, false)

			postReconcileRun, err := clients.Pipeline.TektonV1alpha1().Runs("test").Get(prt.TestAssets.Ctx, runName, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Run get request failed, %v", err)
			}

			gotTimeoutValue := postReconcileRun.GetTimeout()

			if d := cmp.Diff(gotTimeoutValue, tc.wantRunTimeout); d != "" {
				t.Fatalf("Expected timeout for created Run, but got a mismatch %s", diff.PrintWantGot(d))
			}

			if reconciledRun.Status.CompletionTime == nil {
				t.Errorf("Expected a CompletionTime on already timedout PipelineRun but was nil")
			}

			// The PipelineRun should be timed out.
			if reconciledRun.Status.GetCondition(apis.ConditionSucceeded).Reason != "PipelineRunTimeout" {
				t.Errorf("Expected PipelineRun to be timed out, but condition reason is %s",
					reconciledRun.Status.GetCondition(apis.ConditionSucceeded))
			}

			actions := clients.Pipeline.Actions()
			if len(actions) < 2 {
				t.Fatalf("Expected client to have at least two action implementation but it has %d", len(actions))
			}

			// The patch operation to cancel the run must be executed.
			var got []jsonpatch.Operation
			for _, a := range actions {
				if action, ok := a.(ktesting.PatchAction); ok {
					if a.(ktesting.PatchAction).Matches("patch", "runs") {
						err := json.Unmarshal(action.GetPatch(), &got)
						if err != nil {
							t.Fatalf("Expected to get a patch operation for cancel,"+
								" but got error: %v\n", err)
						}
						break
					}
				}
			}
			want := []jsonpatch.JsonPatchOperation{{
				Operation: "add",
				Path:      "/spec/status",
				Value:     "RunCancelled",
			}}
			if d := cmp.Diff(got, want); d != "" {
				t.Fatalf("Expected RunCancelled patch operation, but got a mismatch %s", diff.PrintWantGot(d))
			}
		})
	}
}

func TestReconcileWithTimeoutDeprecated(t *testing.T) {
	// TestReconcileWithTimeoutDeprecated runs "Reconcile" on a PipelineRun that has timed out.
	// It verifies that reconcile is successful, the pipeline status updated and events generated.
	ps := []*v1beta1.Pipeline{simpleHelloWorldPipeline}
	prs := []*v1beta1.PipelineRun{parse.MustParsePipelineRun(t, `
metadata:
  name: test-pipeline-run-with-timeout
  namespace: foo
spec:
  pipelineRef:
    name: test-pipeline
  serviceAccountName: test-sa
  timeout: 12h0m0s
status:
  startTime: "2021-12-31T00:00:00Z"
`)}
	ts := []*v1beta1.Task{simpleHelloWorldTask}

	d := test.Data{
		PipelineRuns: prs,
		Pipelines:    ps,
		Tasks:        ts,
	}
	prt := newPipelineRunTest(d, t)
	defer prt.Cancel()

	wantEvents := []string{
		"Warning Failed PipelineRun \"test-pipeline-run-with-timeout\" failed to finish within \"12h0m0s\"",
	}
	reconciledRun, clients := prt.reconcileRun("foo", "test-pipeline-run-with-timeout", wantEvents, false)

	if reconciledRun.Status.CompletionTime == nil {
		t.Errorf("Expected a CompletionTime on invalid PipelineRun but was nil")
	}

	// The PipelineRun should be timed out.
	if reconciledRun.Status.GetCondition(apis.ConditionSucceeded).Reason != "PipelineRunTimeout" {
		t.Errorf("Expected PipelineRun to be timed out, but condition reason is %s", reconciledRun.Status.GetCondition(apis.ConditionSucceeded))
	}

	// Check that the expected TaskRun was created
	actual := getTaskRunCreations(t, clients.Pipeline.Actions(), 2)[0]

	// The TaskRun timeout should be less than or equal to the PipelineRun timeout.
	if actual.Spec.Timeout.Duration > prs[0].Spec.Timeout.Duration {
		t.Errorf("TaskRun timeout %s should be less than or equal to PipelineRun timeout %s", actual.Spec.Timeout.Duration.String(), prs[0].Spec.Timeout.Duration.String())
	}
}

func TestReconcileWithTimeouts(t *testing.T) {
	// TestReconcileWithTimeouts runs "Reconcile" on a PipelineRun that has timed out.
	// It verifies that reconcile is successful, the pipeline status updated and events generated.
	ps := []*v1beta1.Pipeline{simpleHelloWorldPipeline}
	prs := []*v1beta1.PipelineRun{parse.MustParsePipelineRun(t, `
metadata:
  name: test-pipeline-run-with-timeout
  namespace: foo
spec:
  pipelineRef:
    name: test-pipeline
  serviceAccountName: test-sa
  timeouts:
    pipeline: 12h0m0s
status:
  startTime: "2021-12-31T00:00:00Z"
`)}
	ts := []*v1beta1.Task{simpleHelloWorldTask}

	d := test.Data{
		PipelineRuns: prs,
		Pipelines:    ps,
		Tasks:        ts,
	}
	prt := newPipelineRunTest(d, t)
	defer prt.Cancel()

	wantEvents := []string{
		"Warning Failed PipelineRun \"test-pipeline-run-with-timeout\" failed to finish within \"12h0m0s\"",
	}
	reconciledRun, clients := prt.reconcileRun("foo", "test-pipeline-run-with-timeout", wantEvents, false)

	if reconciledRun.Status.CompletionTime == nil {
		t.Errorf("Expected a CompletionTime on invalid PipelineRun but was nil")
	}

	// The PipelineRun should be timed out.
	if reconciledRun.Status.GetCondition(apis.ConditionSucceeded).Reason != "PipelineRunTimeout" {
		t.Errorf("Expected PipelineRun to be timed out, but condition reason is %s", reconciledRun.Status.GetCondition(apis.ConditionSucceeded))
	}

	// Check that the expected TaskRun was created
	actual := getTaskRunCreations(t, clients.Pipeline.Actions(), 2)[0]

	// The TaskRun timeout should be less than or equal to the PipelineRun timeout.
	if actual.Spec.Timeout.Duration > prs[0].Spec.Timeouts.Pipeline.Duration {
		t.Errorf("TaskRun timeout %s should be less than or equal to PipelineRun timeout %s", actual.Spec.Timeout.Duration.String(), prs[0].Spec.Timeouts.Pipeline.Duration.String())
	}
}
