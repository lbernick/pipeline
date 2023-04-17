package affinityassistant_test

import (
	"context"
	"testing"

	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/internal/affinityassistant"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing" // Setup system.Namespace()
)

const (
	featureFlagDisableAffinityAssistantKey = "disable-affinity-assistant" // FIXME
)

func TestGetAffinityAssistantBehavior(t *testing.T) {
	for _, tc := range []struct {
		description string
		configMap   *corev1.ConfigMap
		expected    affinityassistant.AffinityAssitantBehavior
	}{{
		description: "Default behaviour: A missing disable-affinity-assistant flag should result in false",
		configMap: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: config.GetFeatureFlagsConfigName(), Namespace: system.Namespace()},
			Data:       map[string]string{},
		},
		expected: affinityassistant.AffinityAssistantPerWorkspace,
	}, {
		description: "Setting disable-affinity-assistant to false should result in false",
		configMap: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: config.GetFeatureFlagsConfigName(), Namespace: system.Namespace()},
			Data: map[string]string{
				featureFlagDisableAffinityAssistantKey: "false",
			},
		},
		expected: affinityassistant.AffinityAssistantPerWorkspace,
	}, {
		description: "Setting disable-affinity-assistant to true should result in true",
		configMap: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: config.GetFeatureFlagsConfigName(), Namespace: system.Namespace()},
			Data: map[string]string{
				featureFlagDisableAffinityAssistantKey: "true",
			},
		},
		expected: affinityassistant.AffinityAssistantDisabled,
	}} {
		t.Run(tc.description, func(t *testing.T) {
			// c := Reconciler{
			// 	KubeClientSet: fakek8s.NewSimpleClientset(
			// 		tc.configMap,
			// 	),
			// 	Images: pipeline.Images{},
			// }
			store := config.NewStore(logtesting.TestLogger(t))
			store.OnConfigChanged(tc.configMap)
			if result := affinityassistant.GetAffinityAssistantBehavior(store.ToContext(context.Background())); result != tc.expected {
				t.Errorf("Expected %s Received %s", tc.expected, result)
			}
		})
	}
}
