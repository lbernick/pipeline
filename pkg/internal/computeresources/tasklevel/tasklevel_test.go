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

package tasklevel_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/internal/computeresources/tasklevel"
	"github.com/tektoncd/pipeline/test/diff"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestPodBuild_TaskLevelResourceRequirements(t *testing.T) {
	testcases := []struct {
		desc                     string
		ts                       v1beta1.TaskSpec
		trs                      v1beta1.TaskRunSpec
		expectedComputeResources []corev1.ResourceRequirements
	}{{
		desc: "only with requests",
		ts: v1beta1.TaskSpec{
			Steps: []v1beta1.Step{{
				Name:    "1st-step",
				Image:   "image",
				Command: []string{"cmd"},
			}, {
				Name:    "2nd-step",
				Image:   "image",
				Command: []string{"cmd"},
			}},
		},
		trs: v1beta1.TaskRunSpec{
			ComputeResources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")},
			},
		},
		expectedComputeResources: []corev1.ResourceRequirements{{
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
		}, {
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
		}},
	}, {
		desc: "only with limits",
		ts: v1beta1.TaskSpec{
			Steps: []v1beta1.Step{{
				Name:    "1st-step",
				Image:   "image",
				Command: []string{"cmd"},
			}, {
				Name:    "2nd-step",
				Image:   "image",
				Command: []string{"cmd"},
			}},
		},
		trs: v1beta1.TaskRunSpec{
			ComputeResources: &corev1.ResourceRequirements{
				Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m")},
			},
		},
		expectedComputeResources: []corev1.ResourceRequirements{{
			Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m")},
		}, {
			Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m")},
		}},
	}, {
		desc: "both with requests and limits",
		ts: v1beta1.TaskSpec{
			Steps: []v1beta1.Step{{
				Name:    "1st-step",
				Image:   "image",
				Command: []string{"cmd"},
			}, {
				Name:    "2nd-step",
				Image:   "image",
				Command: []string{"cmd"},
			}},
		},
		trs: v1beta1.TaskRunSpec{
			ComputeResources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
				Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")},
			},
		},
		expectedComputeResources: []corev1.ResourceRequirements{{
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m")},
			Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")},
		}, {
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m")},
			Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")},
		}},
	}, {
		desc: "with sidecar resource requirements",
		ts: v1beta1.TaskSpec{
			Steps: []v1beta1.Step{{
				Name:    "1st-step",
				Image:   "image",
				Command: []string{"cmd"},
			}},
			Sidecars: []v1beta1.Sidecar{{
				Name: "sidecar",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("750m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("1.5"),
					},
				},
			}},
		},
		trs: v1beta1.TaskRunSpec{
			ComputeResources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
			},
		},
		expectedComputeResources: []corev1.ResourceRequirements{{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
		}, {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("750m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("1.5"),
			},
		}},
	}}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			tasklevel.ApplyTaskLevelComputeResources(tc.ts.Steps, tc.trs.ComputeResources)

			if err := verifyTaskLevelComputeResources(tc.ts.Steps, tc.ts.Sidecars, tc.expectedComputeResources); err != nil {
				t.Errorf("verifyTaskLevelComputeResources: %v", err)
			}
		})
	}
}

// verifyTaskLevelComputeResources verifies that the given TaskRun's containers have the expected compute resources.
func verifyTaskLevelComputeResources(steps []v1beta1.Step, sidecars []v1beta1.Sidecar, expectedComputeResources []corev1.ResourceRequirements) error {
	if len(expectedComputeResources) != len(steps)+len(sidecars) {
		return fmt.Errorf("expected %d compute resource requirements, got %d", len(expectedComputeResources), len(steps))
	}
	i := 0
	for _, step := range steps {
		if d := cmp.Diff(expectedComputeResources[i], step.Resources); d != "" {
			return fmt.Errorf("container \"#%d\" resource requirements don't match %s", i, diff.PrintWantGot(d))
		}
		i++
	}
	for _, sidecar := range sidecars {
		if d := cmp.Diff(expectedComputeResources[i], sidecar.Resources); d != "" {
			return fmt.Errorf("container \"#%d\" resource requirements don't match %s", i, diff.PrintWantGot(d))
		}
		i++
	}
	return nil
}
