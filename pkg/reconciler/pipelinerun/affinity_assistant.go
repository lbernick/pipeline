/*
Copyright 2020 The Tekton Authors

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
	"crypto/sha256"
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/internal/affinityassistant"
	"github.com/tektoncd/pipeline/pkg/reconciler/volumeclaim"
	"github.com/tektoncd/pipeline/pkg/workspace"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/logging"
)

const (
	// ReasonCouldntCreateAffinityAssistantStatefulSet indicates that a PipelineRun uses workspaces with PersistentVolumeClaim
	// as a volume source and expect an Assistant StatefulSet, but couldn't create a StatefulSet.
	ReasonCouldntCreateAffinityAssistantStatefulSet = "CouldntCreateAffinityAssistantStatefulSet"
)

// createAffinityAssistantsPerWorkspace creates an Affinity Assistant StatefulSet for every workspace in the PipelineRun that
// use a PersistentVolumeClaim volume. This is done to achieve Node Affinity for all TaskRuns that
// share the workspace volume and make it possible for the tasks to execute parallel while sharing volume.
func (c *Reconciler) createAffinityAssistantsPerWorkspace(ctx context.Context, wb []v1beta1.WorkspaceBinding, pr *v1beta1.PipelineRun, namespace string) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContextOrDefaults(ctx)

	var errs []error
	for _, w := range wb {
		var claimTemplates []corev1.PersistentVolumeClaim
		var claims []corev1.PersistentVolumeClaimVolumeSource
		if w.PersistentVolumeClaim != nil {
			claims = append(claims, *w.PersistentVolumeClaim.DeepCopy())
		} else if w.VolumeClaimTemplate != nil {
			claimTemplate := w.VolumeClaimTemplate.DeepCopy()
			claimTemplate.Name = volumeclaim.GetPersistentVolumeClaimName(w.VolumeClaimTemplate, w, *kmeta.NewControllerRef(pr))
			claimTemplates = append(claimTemplates, *claimTemplate)
		} else {
			continue
		}

		if w.PersistentVolumeClaim != nil || w.VolumeClaimTemplate != nil {
			affinityAssistantName := getPerWorkspaceAffinityAssistantName(w.Name, pr.Name)
			_, err := c.KubeClientSet.AppsV1().StatefulSets(namespace).Get(ctx, affinityAssistantName, metav1.GetOptions{})
			switch {
			case apierrors.IsNotFound(err):
				affinityAssistantStatefulSet := affinityAssistantStatefulSet(affinityAssistantName, pr, claimTemplates, claims, c.Images.NopImage, cfg.Defaults.DefaultAAPodTemplate)
				_, err := c.KubeClientSet.AppsV1().StatefulSets(namespace).Create(ctx, affinityAssistantStatefulSet, metav1.CreateOptions{})
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to create StatefulSet %s: %w", affinityAssistantName, err))
				}
				if err == nil {
					logger.Infof("Created StatefulSet %s in namespace %s", affinityAssistantName, namespace)
				}
			case err != nil:
				errs = append(errs, fmt.Errorf("failed to retrieve StatefulSet %s: %w", affinityAssistantName, err))
			}
		}
	}
	return errorutils.NewAggregate(errs)
}

// createAffinityAssistantsPerPipelineRun TODO
func (c *Reconciler) createAffinityAssistantsPerPipelineRun(ctx context.Context, wb []v1beta1.WorkspaceBinding, pr *v1beta1.PipelineRun, namespace string) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContextOrDefaults(ctx)
	affinityAssistantName := getPerPipelineRunAffinityAssistantName(pr.Name)
	_, err := c.KubeClientSet.AppsV1().StatefulSets(namespace).Get(ctx, affinityAssistantName, metav1.GetOptions{})
	var claimTemplates []corev1.PersistentVolumeClaim
	var claims []corev1.PersistentVolumeClaimVolumeSource
	for _, w := range wb {
		if w.PersistentVolumeClaim != nil {
			claims = append(claims, *w.PersistentVolumeClaim.DeepCopy())
		} else if w.VolumeClaimTemplate != nil {
			claimTemplate := w.VolumeClaimTemplate.DeepCopy()
			ownerRef := kmeta.NewControllerRef(pr)
			if len(claimTemplate.Labels) == 0 {
				claimTemplate.Labels = make(map[string]string)
			}
			claimTemplate.Labels["tekton.dev/workspace"] = w.Name
			claimTemplate.OwnerReferences = []metav1.OwnerReference{*ownerRef}
			claimTemplate.Name = volumeclaim.GetPersistentVolumeClaimName(w.VolumeClaimTemplate, w, *ownerRef)
			claimTemplates = append(claimTemplates, *claimTemplate)
		}

	}
	var errs []error
	switch {
	case apierrors.IsNotFound(err):
		affinityAssistantStatefulSet := affinityAssistantStatefulSet(affinityAssistantName, pr, claimTemplates, claims, c.Images.NopImage, cfg.Defaults.DefaultAAPodTemplate)
		_, err := c.KubeClientSet.AppsV1().StatefulSets(namespace).Create(ctx, affinityAssistantStatefulSet, metav1.CreateOptions{})
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to create StatefulSet %s: %w", affinityAssistantName, err))
		}
		if err == nil {
			logger.Infof("Created StatefulSet %s in namespace %s", affinityAssistantName, namespace)
		}
	case err != nil:
		errs = append(errs, fmt.Errorf("failed to retrieve StatefulSet %s: %w", affinityAssistantName, err))
	}

	return errorutils.NewAggregate(errs)
}

func (c *Reconciler) cleanupAffinityAssistants(ctx context.Context, pr *v1beta1.PipelineRun) error {
	// omit cleanup if the feature is disabled
	affinityAssistantBehavior := affinityassistant.GetAffinityAssistantBehavior(ctx)
	switch affinityAssistantBehavior {
	case affinityassistant.AffinityAssistantDisabled:
		return nil
	case affinityassistant.AffinityAssistantPerWorkspace:
		var errs []error
		for _, w := range pr.Spec.Workspaces {
			if w.PersistentVolumeClaim != nil || w.VolumeClaimTemplate != nil {
				affinityAssistantStsName := getPerWorkspaceAffinityAssistantName(w.Name, pr.Name)
				if err := c.KubeClientSet.AppsV1().StatefulSets(pr.Namespace).Delete(ctx, affinityAssistantStsName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
					errs = append(errs, fmt.Errorf("failed to delete StatefulSet %s: %w", affinityAssistantStsName, err))
				}
			}
		}
		return errorutils.NewAggregate(errs)
	case affinityassistant.AffinityAssistantPerPipelineRun:
		// TODO: Clean up PVCs as well
		affinityAssistantStsName := getPerPipelineRunAffinityAssistantName(pr.Name)
		if err := c.KubeClientSet.AppsV1().StatefulSets(pr.Namespace).Delete(ctx, affinityAssistantStsName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete StatefulSet %s: %w", affinityAssistantStsName, err)
		}
	default:
		return fmt.Errorf("invalid affinity assistant behavior %s: ", affinityAssistantBehavior)
	}
	return nil
}

func getPerWorkspaceAffinityAssistantName(pipelineWorkspaceName string, pipelineRunName string) string {
	hashBytes := sha256.Sum256([]byte(pipelineWorkspaceName + pipelineRunName))
	hashString := fmt.Sprintf("%x", hashBytes)
	return fmt.Sprintf("%s-%s", "affinity-assistant", hashString[:10])
}

func getPerPipelineRunAffinityAssistantName(pipelineRunName string) string {
	hashBytes := sha256.Sum256([]byte(pipelineRunName))
	hashString := fmt.Sprintf("%x", hashBytes)
	return fmt.Sprintf("%s-%s", "affinity-assistant", hashString[:10])
}

func getStatefulSetLabels(pr *v1beta1.PipelineRun, affinityAssistantName string) map[string]string {
	// Propagate labels from PipelineRun to StatefulSet.
	labels := make(map[string]string, len(pr.ObjectMeta.Labels)+1)
	for key, val := range pr.ObjectMeta.Labels {
		labels[key] = val
	}
	labels[pipeline.PipelineRunLabelKey] = pr.Name

	// LabelInstance is used to configure PodAffinity for all TaskRuns belonging to this Affinity Assistant
	// LabelComponent is used to configure PodAntiAffinity to other Affinity Assistants
	labels[workspace.LabelInstance] = affinityAssistantName
	labels[workspace.LabelComponent] = workspace.ComponentNameAffinityAssistant
	return labels
}

func affinityAssistantStatefulSet(name string, pr *v1beta1.PipelineRun, claimTemplates []corev1.PersistentVolumeClaim, claims []corev1.PersistentVolumeClaimVolumeSource, affinityAssistantImage string, defaultAATpl *pod.AffinityAssistantTemplate) *appsv1.StatefulSet {
	// We want a singleton pod
	replicas := int32(1)

	tpl := &pod.AffinityAssistantTemplate{}
	// merge pod template from spec and default if any of them are defined
	if pr.Spec.PodTemplate != nil || defaultAATpl != nil {
		tpl = pod.MergeAAPodTemplateWithDefault(pr.Spec.PodTemplate.ToAffinityAssistantTemplate(), defaultAATpl)
	}

	var mounts []corev1.VolumeMount
	for _, claimTemplate := range claimTemplates {
		mounts = append(mounts, corev1.VolumeMount{Name: claimTemplate.Name, MountPath: claimTemplate.Name})
	}

	containers := []corev1.Container{{
		Name:  "affinity-assistant",
		Image: affinityAssistantImage,
		Args:  []string{"tekton_run_indefinitely"},

		// Set requests == limits to get QoS class _Guaranteed_.
		// See https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/#create-a-pod-that-gets-assigned-a-qos-class-of-guaranteed
		// Affinity Assistant pod is a placeholder; request minimal resources
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"cpu":    resource.MustParse("50m"),
				"memory": resource.MustParse("100Mi"),
			},
			Requests: corev1.ResourceList{
				"cpu":    resource.MustParse("50m"),
				"memory": resource.MustParse("100Mi"),
			},
		},
		VolumeMounts: mounts,
	}}

	// TODO: If WS specifies persistent volume claim, include as volumes in pod template. Will this require setting storage class???
	// Otherwise, include as volumeclaimtemplate on SS and volume mounts on pod

	var volumes []corev1.Volume
	for i, claim := range claims {
		volumes = append(volumes, corev1.Volume{
			Name: fmt.Sprintf("workspace-%d", i),
			VolumeSource: corev1.VolumeSource{
				// A Pod mounting a PersistentVolumeClaim that has a StorageClass with
				// volumeBindingMode: Immediate
				// the PV is allocated on a Node first, and then the pod need to be
				// scheduled to that node.
				// To support those PVCs, the Affinity Assistant must also mount the
				// same PersistentVolumeClaim - to be sure that the Affinity Assistant
				// pod is scheduled to the same Availability Zone as the PV, when using
				// a regional cluster. This is called VolumeScheduling.
				PersistentVolumeClaim: claim.DeepCopy(),
			},
		})
	}

	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Labels:          getStatefulSetLabels(pr, name),
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(pr)},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: getStatefulSetLabels(pr, name),
			},
			VolumeClaimTemplates: claimTemplates, // TODO: These volumes don't get deleted when the statefulset is deleted!
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: getStatefulSetLabels(pr, name),
				},
				Spec: corev1.PodSpec{
					Containers: containers,

					Tolerations:      tpl.Tolerations,
					NodeSelector:     tpl.NodeSelector,
					ImagePullSecrets: tpl.ImagePullSecrets,

					Affinity: getAssistantAffinityMergedWithPodTemplateAffinity(pr),
					Volumes:  volumes,
				},
			},
		},
	}
}

// getAssistantAffinityMergedWithPodTemplateAffinity return the affinity that merged with PipelineRun PodTemplate affinity.
func getAssistantAffinityMergedWithPodTemplateAffinity(pr *v1beta1.PipelineRun) *corev1.Affinity {
	// use podAntiAffinity to repel other affinity assistants
	repelOtherAffinityAssistantsPodAffinityTerm := corev1.WeightedPodAffinityTerm{
		Weight: 100,
		PodAffinityTerm: corev1.PodAffinityTerm{
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					workspace.LabelComponent: workspace.ComponentNameAffinityAssistant,
				},
			},
			TopologyKey: "kubernetes.io/hostname",
		},
	}

	affinityAssistantsAffinity := &corev1.Affinity{}
	if pr.Spec.PodTemplate != nil && pr.Spec.PodTemplate.Affinity != nil {
		affinityAssistantsAffinity = pr.Spec.PodTemplate.Affinity
	}
	if affinityAssistantsAffinity.PodAntiAffinity == nil {
		affinityAssistantsAffinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
	}
	affinityAssistantsAffinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution =
		append(affinityAssistantsAffinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
			repelOtherAffinityAssistantsPodAffinityTerm)

	return affinityAssistantsAffinity
}
