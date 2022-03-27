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

package v1beta1

import (
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/kmeta"
)

const (
	// TaskRunResultType default task run result value
	TaskRunResultType ResultType = 1
	// PipelineResourceResultType default pipeline result value
	PipelineResourceResultType = 2
	// InternalTektonResultType default internal tekton result value
	InternalTektonResultType = 3
	// UnknownResultType default unknown result type value
	UnknownResultType = 10
)

// +genclient
// +genclient:noStatus
// +genreconciler:krshapedlogic=false
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Task represents a collection of sequential steps that are run as part of a
// Pipeline using a set of inputs and producing a set of outputs. Tasks execute
// when TaskRuns are created that provide the input parameters and resources and
// output resources the Task requires.
//
// +k8s:openapi-gen=true
type Task struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the Task from the client
	// +optional
	Spec TaskSpec `json:"spec"`
}

var _ kmeta.OwnerRefable = (*Task)(nil)

// TaskSpec returns the task's spec
func (t *Task) TaskSpec() TaskSpec {
	return t.Spec
}

// TaskMetadata returns the task's ObjectMeta
func (t *Task) TaskMetadata() metav1.ObjectMeta {
	return t.ObjectMeta
}

// Copy returns a deep copy of the task
func (t *Task) Copy() TaskObject {
	return t.DeepCopy()
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*Task) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind(pipeline.TaskControllerName)
}

// TaskSpec defines the desired state of Task.
type TaskSpec struct {
	// Resources is a list input and output resource to run the task
	// Resources are represented in TaskRuns as bindings to instances of
	// PipelineResources.
	// +optional
	Resources *TaskResources `json:"resources,omitempty"`

	// Params is a list of input parameters required to run the task. Params
	// must be supplied as inputs in TaskRuns unless they declare a default
	// value.
	// +optional
	Params []ParamSpec `json:"params,omitempty"`

	// Description is a user-facing description of the task that may be
	// used to populate a UI.
	// +optional
	Description string `json:"description,omitempty"`

	// Steps are the steps of the build; each step is run sequentially with the
	// source mounted into /workspace.
	Steps []Step `json:"steps,omitempty"`

	// Volumes is a collection of volumes that are available to mount into the
	// steps of the build.
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// StepTemplate can be used as the basis for all step containers within the
	// Task, so that the steps inherit settings on the base container.
	StepTemplate *Container `json:"stepTemplate,omitempty"`

	// Sidecars are run alongside the Task's step containers. They begin before
	// the steps start and end after the steps complete.
	Sidecars []Sidecar `json:"sidecars,omitempty"`

	// Workspaces are the volumes that this Task requires.
	Workspaces []WorkspaceDeclaration `json:"workspaces,omitempty"`

	// Results are values that this Task can output
	Results []TaskResult `json:"results,omitempty"`
}

// TaskResult used to describe the results of a task
type TaskResult struct {
	// Name the given name
	Name string `json:"name"`

	// Description is a human-readable description of the result
	// +optional
	Description string `json:"description"`
}

// Step embeds the Container type, which allows it to include fields not
// provided by Container.
type Step struct {
	Container `json:",inline"`

	// Script is the contents of an executable file to execute.
	//
	// If Script is not empty, the Step cannot have an Command and the Args will be passed to the Script.
	// +optional
	Script string `json:"script,omitempty"`

	// Timeout is the time after which the step times out. Defaults to never.
	// Refer to Go's ParseDuration documentation for expected format: https://golang.org/pkg/time/#ParseDuration
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// This is an alpha field. You must set the "enable-api-fields" feature flag to "alpha"
	// for this field to be supported.
	//
	// Workspaces is a list of workspaces from the Task that this Step wants
	// exclusive access to. Adding a workspace to this list means that any
	// other Step or Sidecar that does not also request this Workspace will
	// not have access to it.
	// +optional
	Workspaces []WorkspaceUsage `json:"workspaces,omitempty"`

	// OnError defines the exiting behavior of a container on error
	// can be set to [ continue | stopAndFail ]
	// stopAndFail indicates exit the taskRun if the container exits with non-zero exit code
	// continue indicates continue executing the rest of the steps irrespective of the container exit code
	OnError string `json:"onError,omitempty"`
}

// Sidecar has nearly the same data structure as Step, consisting of a Container and an optional Script, but does not have the ability to timeout.
type Sidecar struct {
	Container `json:",inline"`

	// Script is the contents of an executable file to execute.
	//
	// If Script is not empty, the Step cannot have an Command or Args.
	// +optional
	Script string `json:"script,omitempty"`

	// This is an alpha field. You must set the "enable-api-fields" feature flag to "alpha"
	// for this field to be supported.
	//
	// Workspaces is a list of workspaces from the Task that this Sidecar wants
	// exclusive access to. Adding a workspace to this list means that any
	// other Step or Sidecar that does not also request this Workspace will
	// not have access to it.
	// +optional
	Workspaces []WorkspaceUsage `json:"workspaces,omitempty"`
}

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Container struct {
	// Name of the container specified as a DNS_LABEL.
	// Each container in a pod must have a unique name (DNS_LABEL).
	// Cannot be updated.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Docker image name.
	// More info: https://kubernetes.io/docs/concepts/containers/images
	// This field is optional to allow higher level config management to default or override
	// container images in workload controllers like Deployments and StatefulSets.
	// +optional
	Image string `json:"image,omitempty" protobuf:"bytes,2,opt,name=image"`
	// Entrypoint array. Not executed within a shell.
	// The docker image's ENTRYPOINT is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced
	// to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will
	// produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless
	// of whether the variable exists or not. Cannot be updated.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	Command []string `json:"command,omitempty" protobuf:"bytes,3,rep,name=command"`
	// Arguments to the entrypoint.
	// The docker image's CMD is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced
	// to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will
	// produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless
	// of whether the variable exists or not. Cannot be updated.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	Args []string `json:"args,omitempty" protobuf:"bytes,4,rep,name=args"`
	// Container's working directory.
	// If not specified, the container runtime's default will be used, which
	// might be configured in the container image.
	// Cannot be updated.
	// +optional
	WorkingDir string `json:"workingDir,omitempty" protobuf:"bytes,5,opt,name=workingDir"`
	// List of ports to expose from the container. Exposing a port here gives
	// the system additional information about the network connections a
	// container uses, but is primarily informational. Not specifying a port here
	// DOES NOT prevent that port from being exposed. Any port which is
	// listening on the default "0.0.0.0" address inside a container will be
	// accessible from the network.
	// Cannot be updated.
	// +optional
	// +patchMergeKey=containerPort
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=containerPort
	// +listMapKey=protocol
	Ports []corev1.ContainerPort `json:"ports,omitempty" patchStrategy:"merge" patchMergeKey:"containerPort" protobuf:"bytes,6,rep,name=ports"`
	// List of sources to populate environment variables in the container.
	// The keys defined within a source must be a C_IDENTIFIER. All invalid keys
	// will be reported as an event when the container is starting. When a key exists in multiple
	// sources, the value associated with the last source will take precedence.
	// Values defined by an Env with a duplicate key will take precedence.
	// Cannot be updated.
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty" protobuf:"bytes,19,rep,name=envFrom"`
	// List of environment variables to set in the container.
	// Cannot be updated.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	Env []corev1.EnvVar `json:"env,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,7,rep,name=env"`
	// Compute Resources required by this container.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// +optional
	Resources ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`
	// Pod volumes to mount into the container's filesystem.
	// Cannot be updated.
	// +optional
	// +patchMergeKey=mountPath
	// +patchStrategy=merge
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty" patchStrategy:"merge" patchMergeKey:"mountPath" protobuf:"bytes,9,rep,name=volumeMounts"`
	// volumeDevices is the list of block devices to be used by the container.
	// +patchMergeKey=devicePath
	// +patchStrategy=merge
	// +optional
	VolumeDevices []corev1.VolumeDevice `json:"volumeDevices,omitempty" patchStrategy:"merge" patchMergeKey:"devicePath" protobuf:"bytes,21,rep,name=volumeDevices"`
	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty" protobuf:"bytes,10,opt,name=livenessProbe"`
	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty" protobuf:"bytes,11,opt,name=readinessProbe"`
	// StartupProbe indicates that the Pod has successfully initialized.
	// If specified, no other probes are executed until this completes successfully.
	// If this probe fails, the Pod will be restarted, just as if the livenessProbe failed.
	// This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,
	// when it might take a long time to load data or warm a cache, than during steady-state operation.
	// This cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	StartupProbe *corev1.Probe `json:"startupProbe,omitempty" protobuf:"bytes,22,opt,name=startupProbe"`
	// Actions that the management system should take in response to container lifecycle events.
	// Cannot be updated.
	// +optional
	Lifecycle *corev1.Lifecycle `json:"lifecycle,omitempty" protobuf:"bytes,12,opt,name=lifecycle"`
	// Optional: Path at which the file to which the container's termination message
	// will be written is mounted into the container's filesystem.
	// Message written is intended to be brief final status, such as an assertion failure message.
	// Will be truncated by the node if greater than 4096 bytes. The total message length across
	// all containers will be limited to 12kb.
	// Defaults to /dev/termination-log.
	// Cannot be updated.
	// +optional
	TerminationMessagePath string `json:"terminationMessagePath,omitempty" protobuf:"bytes,13,opt,name=terminationMessagePath"`
	// Indicate how the termination message should be populated. File will use the contents of
	// terminationMessagePath to populate the container status message on both success and failure.
	// FallbackToLogsOnError will use the last chunk of container log output if the termination
	// message file is empty and the container exited with an error.
	// The log output is limited to 2048 bytes or 80 lines, whichever is smaller.
	// Defaults to File.
	// Cannot be updated.
	// +optional
	TerminationMessagePolicy corev1.TerminationMessagePolicy `json:"terminationMessagePolicy,omitempty" protobuf:"bytes,20,opt,name=terminationMessagePolicy,casttype=TerminationMessagePolicy"`
	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/containers/images#updating-images
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty" protobuf:"bytes,14,opt,name=imagePullPolicy,casttype=PullPolicy"`
	// SecurityContext defines the security options the container should be run with.
	// If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty" protobuf:"bytes,15,opt,name=securityContext"`

	// Variables for interactive containers, these have very specialized use-cases (e.g. debugging)
	// and shouldn't be used for general purpose containers.

	// Whether this container should allocate a buffer for stdin in the container runtime. If this
	// is not set, reads from stdin in the container will always result in EOF.
	// Default is false.
	// +optional
	Stdin bool `json:"stdin,omitempty" protobuf:"varint,16,opt,name=stdin"`
	// Whether the container runtime should close the stdin channel after it has been opened by
	// a single attach. When stdin is true the stdin stream will remain open across multiple attach
	// sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the
	// first client attaches to stdin, and then remains open and accepts data until the client disconnects,
	// at which time stdin is closed and remains closed until the container is restarted. If this
	// flag is false, a container processes that reads from stdin will never receive an EOF.
	// Default is false
	// +optional
	StdinOnce bool `json:"stdinOnce,omitempty" protobuf:"varint,17,opt,name=stdinOnce"`
	// Whether this container should allocate a TTY for itself, also requires 'stdin' to be true.
	// Default is false.
	// +optional
	TTY bool `json:"tty,omitempty" protobuf:"varint,18,opt,name=tty"`
}

func (c *Container) ToK8sContainer() (*corev1.Container, error) {
	resources, err := c.Resources.ToK8sResources()
	if err != nil {
		return nil, err
	}
	return &corev1.Container{
		Name:                   c.Name,
		Image:                  c.Image,
		Command:                c.Command,
		Args:                   c.Args,
		WorkingDir:             c.WorkingDir,
		Ports:                  c.Ports,
		EnvFrom:                c.EnvFrom,
		Env:                    c.Env,
		VolumeMounts:           c.VolumeMounts,
		VolumeDevices:          c.VolumeDevices,
		LivenessProbe:          c.LivenessProbe,
		ReadinessProbe:         c.ReadinessProbe,
		StartupProbe:           c.StartupProbe,
		Lifecycle:              c.Lifecycle,
		TerminationMessagePath: c.TerminationMessagePath,
		ImagePullPolicy:        c.ImagePullPolicy,
		SecurityContext:        c.SecurityContext,
		Stdin:                  c.Stdin,
		StdinOnce:              c.StdinOnce,
		TTY:                    c.TTY,
		Resources:              *resources,
	}, nil
}

func (r *ResourceRequirements) ToK8sResources() (*corev1.ResourceRequirements, error) {
	resources := corev1.ResourceRequirements{}
	if r.Requests != nil {
		resources.Requests = make(corev1.ResourceList)
	}
	if r.Limits != nil {
		resources.Limits = make(corev1.ResourceList)
	}
	var invalidRequests []string
	for k, v := range r.Requests {
		q, err := resource.ParseQuantity(v)
		if err != nil {
			invalidRequests = append(invalidRequests, v)
		} else {
			resources.Requests[k] = q
		}
	}
	if len(invalidRequests) > 0 {
		return nil, fmt.Errorf("invalid resource requests %s", invalidRequests)
	}
	var invalidLimits []string
	for k, v := range r.Limits {
		q, err := resource.ParseQuantity(v)
		if err != nil {
			invalidLimits = append(invalidLimits, v)
		} else {
			resources.Limits[k] = q
		}
	}
	if len(invalidLimits) > 0 {
		return nil, fmt.Errorf("invalid resource limits %s", invalidLimits)
	}
	return &resources, nil
}

// ResourceRequirements describes the compute resource requirements.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type ResourceRequirements struct {
	// Limits describes the maximum amount of compute resources allowed.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// +optional
	Limits ResourceList `json:"limits,omitempty" protobuf:"bytes,1,rep,name=limits,casttype=ResourceList,castkey=ResourceName"`
	// Requests describes the minimum amount of compute resources required.
	// If Requests is omitted for a container, it defaults to Limits if that is explicitly specified,
	// otherwise to an implementation-defined value.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// +optional
	Requests ResourceList `json:"requests,omitempty" protobuf:"bytes,2,rep,name=requests,casttype=ResourceList,castkey=ResourceName"`
}

type ResourceList map[corev1.ResourceName]string

// TaskList contains a list of Task
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Task `json:"items"`
}

// TaskRef can be used to refer to a specific instance of a task.
// Copied from CrossVersionObjectReference: https://github.com/kubernetes/kubernetes/blob/169df7434155cbbc22f1532cba8e0a9588e29ad8/pkg/apis/autoscaling/types.go#L64
type TaskRef struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name,omitempty"`
	// TaskKind indicates the kind of the task, namespaced or cluster scoped.
	Kind TaskKind `json:"kind,omitempty"`
	// API version of the referent
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
	// Bundle url reference to a Tekton Bundle.
	// +optional
	Bundle string `json:"bundle,omitempty"`

	// ResolverRef allows referencing a Task in a remote location
	// like a git repo. This field is only supported when the alpha
	// feature gate is enabled.
	// +optional
	ResolverRef `json:",omitempty"`
}

// Check that Pipeline may be validated and defaulted.

// TaskKind defines the type of Task used by the pipeline.
type TaskKind string

const (
	// NamespacedTaskKind indicates that the task type has a namespaced scope.
	NamespacedTaskKind TaskKind = "Task"
	// ClusterTaskKind indicates that task type has a cluster scope.
	ClusterTaskKind TaskKind = "ClusterTask"
)
