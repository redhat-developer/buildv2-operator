// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

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

// BuildRunSpec defines the desired state of BuildRun
type BuildRunSpec struct {
	// BuildRef refers to the Build
	BuildRef *BuildRef `json:"buildRef"`

	// ServiceAccount refers to the kubernetes serviceaccount
	// which is used for resource control.
	// Default serviceaccount will be set if it is empty
	// +optional
	ServiceAccount *ServiceAccount `json:"serviceAccount,omitempty"`

	// Timeout defines the maximum run time of this BuildRun.
	// +optional
	// +kubebuilder:validation:Format=duration
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Params is a list of key/value that could be used
	// to set strategy parameters
	// +optional
	ParamValues []ParamValue `json:"paramValues,omitempty"`

	// Output refers to the location where the generated
	// image would be pushed to. It will overwrite the output image in build spec
	// +optional
	Output *Image `json:"output,omitempty"`

	// State is used for canceling a buildrun (and maybe more later on).
	// +optional
	State BuildRunRequestedState `json:"state,omitempty"`
}

// BuildRunRequestedState defines the buildrun state the user can provide to override whatever is the current state.
type BuildRunRequestedState string

const (
	// BuildRunStateCancel indicates that the user wants to cancel the BuildRun,
	// if not already canceled or terminated
	BuildRunStateCancel = "BuildRunCanceled"

	// BuildRunStatePodEvicted indicates that if the pods got evicted
	// due to some reason. (Probably ran out of ephemeral storage)
	BuildRunStatePodEvicted = "PodEvicted"
)

// SourceResult holds the results emitted from the different sources
type SourceResult struct {
	// Name is the name of source
	Name string `json:"name"`

	// Git holds the results emitted from from the
	// step definition of a git source
	//
	// +optional
	Git *GitSourceResult `json:"git,omitempty"`

	// Bundle holds the results emitted from from the
	// step definition of bundle source
	//
	// +optional
	Bundle *BundleSourceResult `json:"bundle,omitempty"`
}

// BundleSourceResult holds the results emitted from the bundle source
type BundleSourceResult struct {
	// Digest hold the image digest result
	Digest string `json:"digest,omitempty"`
}

// GitSourceResult holds the results emitted from the git source
type GitSourceResult struct {
	// CommitSha holds the commit sha of git source
	CommitSha string `json:"commitSha,omitempty"`

	// CommitAuthor holds the commit author of a git source
	CommitAuthor string `json:"commitAuthor,omitempty"`
}

// Output holds the results emitted from the output step (build-and-push)
type Output struct {
	// Digest holds the digest of output image
	Digest string `json:"digest,omitempty"`

	// Size holds the compressed size of output image
	Size string `json:"size,omitempty"`
}

// BuildRunStatus defines the observed state of BuildRun
type BuildRunStatus struct {
	// Sources holds the results emitted from the step definition
	// of different sources
	//
	// +optional
	Sources []SourceResult `json:"sources"`

	// Output holds the results emitted from step definition of an output
	//
	// +optional
	Output *Output `json:"output"`

	// Conditions holds the latest available observations of a resource's current state.
	Conditions Conditions `json:"conditions,omitempty"`

	// LatestTaskRunRef is the name of the TaskRun responsible for executing this BuildRun.
	//
	// TODO: This should be called something like "TaskRunName"
	//
	// +optional
	LatestTaskRunRef *string `json:"latestTaskRunRef,omitempty"`

	// StartTime is the time the build is actually started.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time the build completed.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// BuildSpec is the Build Spec of this BuildRun.
	// +optional
	BuildSpec *BuildSpec `json:"buildSpec,omitempty"`

	// FailedAt points to the resource where the BuildRun failed
	// +optional
	FailedAt *FailedAt `json:"failedAt,omitempty"`
}

// FailedAt describes the location where the failure happened
type FailedAt struct {
	Pod       string `json:"pod,omitempty"`
	Container string `json:"container,omitempty"`
}

// BuildRef can be used to refer to a specific instance of a Build.
type BuildRef struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name"`
	// API version of the referent
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
}

// ServiceAccount can be used to refer to a specific ServiceAccount.
type ServiceAccount struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	// +optional
	Name *string `json:"name,omitempty"`
	// If generates a new ServiceAccount for the build
	// +optional
	Generate bool `json:"generate,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildRun is the Schema representing an instance of build execution
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=buildruns,scope=Namespaced,shortName=br;brs
// +kubebuilder:printcolumn:name="Succeeded",type="string",JSONPath=".status.conditions[?(@.type==\"Succeeded\")].status",description="The Succeeded status of the BuildRun"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type==\"Succeeded\")].reason",description="The Succeeded reason of the BuildRun"
// +kubebuilder:printcolumn:name="StartTime",type="date",JSONPath=".status.startTime",description="The start time of this BuildRun"
// +kubebuilder:printcolumn:name="CompletionTime",type="date",JSONPath=".status.completionTime",description="The completion time of this BuildRun"
type BuildRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildRunSpec   `json:"spec,omitempty"`
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
	return br.Spec.State == BuildRunStateCancel
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
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" description:"last time the condition transit from one status to another"`

	// The reason for the condition last transition.
	// +optional
	Reason string `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`

	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`
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
