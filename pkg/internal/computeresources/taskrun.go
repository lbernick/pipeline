package computeresources

import (
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// returns a mapping of step name to resource requirements
func GetStepComputeResources(taskRun *v1beta1.TaskRun, taskSpec v1beta1.TaskSpec, limitRange *corev1.LimitRange) map[string]corev1.ResourceRequirements {
	// taskSpec.Steps.Resources
	// taskSpec.StepTemplate.Resources
	// taskRun.computeResources
	// taskRun.stepOverrides[].resources
	// limitrange
	return nil
}

func GetSidecarComputeResources(taskRun *v1beta1.TaskRun, taskSpec v1beta1.TaskSpec, limitRange *corev1.LimitRange) map[string]corev1.ResourceRequirements {
	// taskSpec.Sidecars.Resources
	// taskRun.computeResources
	// taskRun.sidecarOverrides[].resources
	// limitrange
	return nil
}

func GetInitContainerComputeResources(taskRun *v1beta1.TaskRun, taskSpec v1beta1.TaskSpec, limitRange *corev1.LimitRange) corev1.ResourceRequirements {
	// limitrange
	return corev1.ResourceRequirements{}
}
