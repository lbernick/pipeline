package affinityassistant

import (
	"context"

	"github.com/tektoncd/pipeline/pkg/apis/config"
)

type AffinityAssitantBehavior string

const (
	AffinityAssistantDisabled       = AffinityAssitantBehavior("AffinityAssistantDisabled")
	AffinityAssistantPerWorkspace   = AffinityAssitantBehavior("AffinityAssistantPerWorkspace")
	AffinityAssistantPerPipelineRun = AffinityAssitantBehavior("AffinityAssistantPerPipelineRun")
)

// GetAffinityAssistantBehavior returns a TODO
func GetAffinityAssistantBehavior(ctx context.Context) AffinityAssitantBehavior {
	cfg := config.FromContextOrDefaults(ctx)
	if cfg.FeatureFlags.DisableAffinityAssistant {
		return AffinityAssistantDisabled
	}
	//return AffinityAssistantPerWorkspace
	return AffinityAssistantPerPipelineRun
}
