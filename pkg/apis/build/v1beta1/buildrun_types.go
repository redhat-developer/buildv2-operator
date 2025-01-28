// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// BuildRunDomain is the domain used for all labels and annotations for this resource
	BuildRunDomain = "buildrun.shipwright.io"

	// LabelBuildRun is a label key for BuildRuns to define the name of the BuildRun
	LabelBuildRun = BuildRunDomain + "/name"

	// LabelBuildRunGeneration is a label key for BuildRuns to define the generation
	LabelBuildRunGeneration = BuildRunDomain + "/generation"
)

// VulnerabilitySeverity is an enum for the possible values for severity of a vulnerability
type VulnerabilitySeverity string

const (
	// Critical indicates a critical severity
	Critical VulnerabilitySeverity = "critical"

	// High indicates a high severity
	High VulnerabilitySeverity = "high"

	// Medium indicates a medium severity
	Medium VulnerabilitySeverity = "medium"

	// Low indicates a low severity
	Low VulnerabilitySeverity = "low"

	// Unknown indicates an unknown severity
	Unknown VulnerabilitySeverity = "unknown"
)

type ReferencedBuild struct {
	// Spec refers to an embedded build specification
	//
	// +optional
	Spec *BuildSpec `json:"spec,omitempty"`

	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	//
	// +optional
	Name *string `json:"name,omitempty"`
}

// BuildRunSpec defines the desired state of BuildRun
type BuildRunSpec struct {
	// Build refers to an embedded build specification
	// This field is mandatory
	//
	Build ReferencedBuild `json:"build"`

	// Source refers to the location where the source code is,
	// this could only be a local source
	//
	// +optional
	Source *BuildRunSource `json:"source,omitempty"`

	// ServiceAccount refers to the kubernetes serviceaccount
	// which is used for resource control.
	// Default serviceaccount will be set if it is empty
	//
	// +optional
	ServiceAccount *string `json:"serviceAccount,omitempty"`

	// Timeout defines the maximum run time of this BuildRun.
	//
	// +optional
	// +kubebuilder:validation:Format=duration
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Params is a list of key/value that could be used
	// to set strategy parameters
	//
	// +optional
	ParamValues []ParamValue `json:"paramValues,omitempty"`

	// Output refers to the location where the generated
	// image would be pushed to. It will overwrite the output image in build spec
	//
	// +optional
	Output *Image `json:"output,omitempty"`

	// State is used for canceling a buildrun (and maybe more later on).
	//
	// +optional
	State *BuildRunRequestedState `json:"state,omitempty"`

	// Env contains additional environment variables that should be passed to the build container
	//
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Contains information about retention params
	// +optional
	Retention *BuildRunRetention `json:"retention,omitempty"`

	// Volumes contains volume Overrides of the BuildStrategy volumes in case those are allowed
	// to be overridden. Must only contain volumes that exist in the corresponding BuildStrategy
	// +optional
	Volumes []BuildVolume `json:"volumes,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	//
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, the pod's tolerations.
	// +optional
	// +patchMergeKey=Key
	// +patchStrategy=merge
	Tolerations []corev1.Toleration `json:"tolerations,omitempty" patchStrategy:"merge" patchMergeKey:"Key"`
}

// BuildRunRequestedState defines the buildrun state the user can provide to override whatever is the current state.
type BuildRunRequestedState string

// BuildRunRequestedStatePtr returns a pointer to the passed BuildRunRequestedState.
func BuildRunRequestedStatePtr(s BuildRunRequestedState) *BuildRunRequestedState {
	return &s
}

const (
	// BuildRunStateCancel indicates that the user wants to cancel the BuildRun,
	// if not already canceled or terminated
	BuildRunStateCancel = "BuildRunCanceled"

	// BuildRunStateVulnerabilitiesFound indicates that unignored vulnerabilities were found in the image that was built
	BuildRunStateVulnerabilitiesFound = "VulnerabilitiesFound"

	// BuildRunStatePodEvicted indicates that if the pods got evicted
	// due to some reason. (Probably ran out of ephemeral storage)
	BuildRunStatePodEvicted = "PodEvicted"

	// BuildRunStateStepOutOfMemory indicates that a step failed because it went out of memory.
	BuildRunStateStepOutOfMemory = "StepOutOfMemory"
)

// SourceResult holds the results emitted from the different sources
type SourceResult struct {

	// Git holds the results emitted from the
	// source step of type git
	//
	// +optional
	Git *GitSourceResult `json:"git,omitempty"`

	// OciArtifact holds the results emitted from
	// the source step of type ociArtifact
	//
	// +optional
	OciArtifact *OciArtifactSourceResult `json:"ociArtifact,omitempty"`

	// Timestamp holds the timestamp of the source, which
	// depends on the actual source type and could range from
	// being the commit timestamp or the fileystem timestamp
	// of the most recent source file in the working directory
	//
	// +optional
	Timestamp *metav1.Time `json:"timestamp,omitempty"`
}

// OciArtifactSourceResult holds the results emitted from the bundle source
type OciArtifactSourceResult struct {
	// Digest hold the image digest result
	Digest string `json:"digest,omitempty"`
}

// GitSourceResult holds the results emitted from the git source
type GitSourceResult struct {
	// CommitSha holds the commit sha of git source
	CommitSha string `json:"commitSha,omitempty"`

	// CommitAuthor holds the commit author of a git source
	CommitAuthor string `json:"commitAuthor,omitempty"`

	// BranchName holds the default branch name of the git source
	// this will be set only when revision is not specified in Build object
	//
	// +optional
	BranchName string `json:"branchName,omitempty"`
}

// Vulnerability defines a vulnerability by its ID and severity
type Vulnerability struct {
	ID       string                `json:"id,omitempty"`
	Severity VulnerabilitySeverity `json:"severity,omitempty"`
}

// Output holds the information about the container image that the BuildRun built
type Output struct {
	// Digest holds the digest of output image
	//
	// +optional
	Digest string `json:"digest,omitempty"`

	// Size holds the compressed size of output image
	//
	// +optional
	Size int64 `json:"size,omitempty"`

	// Vulnerabilities holds the list of vulnerabilities detected in the image
	//
	// +optional
	Vulnerabilities []Vulnerability `json:"vulnerabilities,omitempty"`
}

// BuildRunStatus defines the observed state of BuildRun
type BuildRunStatus struct {
	// Source holds the results emitted from the source step
	//
	// +optional
	Source *SourceResult `json:"source,omitempty"`

	// Output holds the results emitted from step definition of an output
	//
	// +optional
	Output *Output `json:"output,omitempty"`

	// Conditions holds the latest available observations of a resource's current state.
	Conditions Conditions `json:"conditions,omitempty"`

	// TaskRunName is the name of the TaskRun responsible for executing this BuildRun.
	//
	// +optional
	TaskRunName *string `json:"taskRunName,omitempty"`

	// StartTime is the time the build is actually started.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time the build completed.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// BuildSpec is the Build Spec of this BuildRun.
	// +optional
	BuildSpec *BuildSpec `json:"buildSpec,omitempty"`

	// FailureDetails contains error details that are collected and surfaced from TaskRun
	// +optional
	FailureDetails *FailureDetails `json:"failureDetails,omitempty"`
}

// Location describes the location where the failure happened
type Location struct {
	Pod       string `json:"pod,omitempty"`
	Container string `json:"container,omitempty"`
}

// FailureDetails describes an error while building images
type FailureDetails struct {
	Reason   string    `json:"reason,omitempty"`
	Message  string    `json:"message,omitempty"`
	Location *Location `json:"location,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildRun is the Schema representing an instance of build execution
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:path=buildruns,scope=Namespaced,shortName=br;brs
// +kubebuilder:printcolumn:name="Succeeded",type="string",JSONPath=".status.conditions[?(@.type==\"Succeeded\")].status",description="The Succeeded status of the BuildRun"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type==\"Succeeded\")].reason",description="The Succeeded reason of the BuildRun"
// +kubebuilder:printcolumn:name="StartTime",type="date",JSONPath=".status.startTime",description="The start time of this BuildRun"
// +kubebuilder:printcolumn:name="CompletionTime",type="date",JSONPath=".status.completionTime",description="The completion time of this BuildRun"
type BuildRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildRunSpec   `json:"spec"`
	Status BuildRunStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildRunList contains a list of BuildRun
type BuildRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BuildRun `json:"items"`
}

// IsDone returns true if the BuildRun's status indicates that it is done.
func (br *BuildRun) IsDone() bool {
	c := br.Status.GetCondition(Succeeded)
	return c != nil && c.GetStatus() != corev1.ConditionUnknown
}

// HasStarted returns true if the BuildRun has a valid start time set in its status.
func (br *BuildRun) HasStarted() bool {
	return br.Status.StartTime != nil && !br.Status.StartTime.IsZero()
}

// IsSuccessful returns true if the BuildRun's status indicates that it is done.
func (br *BuildRun) IsSuccessful() bool {
	c := br.Status.GetCondition(Succeeded)
	return c != nil && c.GetStatus() == corev1.ConditionTrue
}

// IsCanceled returns true if the BuildRun's spec status is set to BuildRunCanceled state.
func (br *BuildRun) IsCanceled() bool {
	return br.Spec.State != nil && *br.Spec.State == BuildRunStateCancel
}

// Conditions defines a list of Condition
type Conditions []Condition

// Type used for defining the conditiont Type field flavour
type Type string

const (
	// Succeeded specifies that the resource has finished.
	// For resources that run to completion.
	Succeeded Type = "Succeeded"
)

// Condition defines the required fields for populating
// Build controllers Conditions
type Condition struct {
	// Type of condition
	// +required
	Type Type `json:"type" description:"type of status condition"`

	// Status of the condition, one of True, False, Unknown.
	// +required
	Status corev1.ConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`

	// LastTransitionTime last time the condition transit from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime" description:"last time the condition transit from one status to another"`

	// The reason for the condition last transition.
	Reason string `json:"reason" description:"one-word CamelCase reason for the condition's last transition"`

	// A human readable message indicating details about the transition.
	Message string `json:"message" description:"human-readable message indicating details about last transition"`
}

// BuildRunRetention struct for buildrun cleanup
type BuildRunRetention struct {
	// TTLAfterFailed defines the maximum duration of time the failed buildrun should exist.
	//
	// +optional
	// +kubebuilder:validation:Format=duration
	TTLAfterFailed *metav1.Duration `json:"ttlAfterFailed,omitempty"`
	// TTLAfterSucceeded defines the maximum duration of time the succeeded buildrun should exist.
	//
	// +optional
	// +kubebuilder:validation:Format=duration
	TTLAfterSucceeded *metav1.Duration `json:"ttlAfterSucceeded,omitempty"`
}

func init() {
	SchemeBuilder.Register(&BuildRun{}, &BuildRunList{})
}

// GetReason returns the condition Reason, it ensures that by getting the Reason
// the call will not panic if the Condition is not present
func (c *Condition) GetReason() string {
	if c == nil {
		return ""
	}
	return c.Reason
}

// GetMessage returns the condition Message, it ensures that by getting the Message
// the call will not panic if the Condition is not present
func (c *Condition) GetMessage() string {
	if c == nil {
		return ""
	}
	return c.Message
}

// GetStatus returns the condition Status, it ensures that by getting the Status
// the call will not panic if the Condition is not present
func (c *Condition) GetStatus() corev1.ConditionStatus {
	if c == nil {
		return ""
	}
	return c.Status
}

// GetCondition returns a condition based on a type from a list of Conditions
func (brs *BuildRunStatus) GetCondition(t Type) *Condition {
	for _, c := range brs.Conditions {
		if c.Type == t {
			return &c
		}
	}
	return nil
}

// IsFailed returns a condition with a False Status
// based on a type from a list of Conditions.
func (brs *BuildRunStatus) IsFailed(t Type) bool {
	for _, c := range brs.Conditions {
		if c.Type == t {
			return c.Status == corev1.ConditionFalse
		}
	}
	return false
}

// SetCondition updates a list of conditions with the provided condition
func (brs *BuildRunStatus) SetCondition(condition *Condition) {
	var idx = -1
	for i, c := range brs.Conditions {
		if c.Type == condition.Type {
			idx = i
			break
		}
	}

	if idx != -1 {
		brs.Conditions[idx] = *condition
	} else {
		brs.Conditions = append(brs.Conditions, *condition)
	}
}

// BuildName returns the name of the associated build, which can be a referenced
// build resource or an embedded build specification
func (buildrunSpec *BuildRunSpec) BuildName() string {
	if buildrunSpec.Build.Name != nil {
		return *buildrunSpec.Build.Name
	}

	// Only BuildRuns with a ReferencedBuild can actually return a proper Build name
	return ""
}
