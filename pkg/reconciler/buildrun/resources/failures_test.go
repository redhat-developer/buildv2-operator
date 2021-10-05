package resources

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
)

var _ = Describe("Surfacing errors", func() {
	Context("resources.UpdateBuildRunUsingTaskFailures", func() {

		It("surfaces errors to BuildRun in failed TaskRun", func() {
			redTaskRun := v1beta1.TaskRun{}
			redTaskRun.Status.Conditions = append(redTaskRun.Status.Conditions,
				apis.Condition{Type: apis.ConditionSucceeded, Reason: v1beta1.TaskRunReasonFailed.String()})
			failedStep := v1beta1.StepState{}

			errorReasonValue := "val1"
			errorMessageValue := "val2"
			errorReasonKey := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorReason)
			errorMessageKey := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorMessage)

			errorReason := v1beta1.PipelineResourceResult{Key: errorReasonKey, Value: errorReasonValue}
			errorMessage := v1beta1.PipelineResourceResult{Key: errorMessageKey, Value: errorMessageValue}
			unrelated := v1beta1.PipelineResourceResult{Key: "unrelated", Value: "unrelated"}

			message, _ := json.Marshal([]v1beta1.PipelineResourceResult{errorReason, errorMessage, unrelated})

			failedStep.Terminated = &v1.ContainerStateTerminated{Message: string(message)}

			redTaskRun.Status.Steps = append(redTaskRun.Status.Steps, failedStep)
			redBuild := v1alpha1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(&redBuild, &redTaskRun)

			Expect(redBuild.Status.Failure.Message).To(Equal(errorMessageValue))
			Expect(redBuild.Status.Failure.Reason).To(Equal(errorReasonValue))
		})

		It("failed TaskRun without error reason and message", func() {
			redTaskRun := v1beta1.TaskRun{}
			redTaskRun.Status.Conditions = append(redTaskRun.Status.Conditions,
				apis.Condition{Type: apis.ConditionSucceeded, Reason: v1beta1.TaskRunReasonFailed.String()})
			failedStep := v1beta1.StepState{}

			unrelated := v1beta1.PipelineResourceResult{Key: "unrelated", Value: "unrelated"}

			message, _ := json.Marshal([]v1beta1.PipelineResourceResult{unrelated})

			failedStep.Terminated = &v1.ContainerStateTerminated{Message: string(message)}

			redTaskRun.Status.Steps = append(redTaskRun.Status.Steps, failedStep)
			redBuild := v1alpha1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(&redBuild, &redTaskRun)

			Expect(redBuild.Status.Failure).To(BeNil())
		})

		It("no errors present in BuildRun for successful TaskRun", func() {
			greenTaskRun := v1beta1.TaskRun{}
			greenTaskRun.Status.Conditions = append(greenTaskRun.Status.Conditions, apis.Condition{Type: apis.ConditionSucceeded})
			greenBuildRun := v1alpha1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(&greenBuildRun, &greenTaskRun)

			Expect(greenBuildRun.Status.Failure).To(BeNil())
		})

		It("TaskRun has not condition succeeded so nothing to do", func() {
			unfinishedTaskRun := v1beta1.TaskRun{}
			unfinishedTaskRun.Status.Conditions = append(unfinishedTaskRun.Status.Conditions, apis.Condition{Type: apis.ConditionReady})
			unfinishedBuildRun := v1alpha1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(&unfinishedBuildRun, &unfinishedTaskRun)
			Expect(unfinishedBuildRun.Status.Failure).To(BeNil())
		})

		It("TaskRun has a unknown reason", func() {
			unknownTaskRun := v1beta1.TaskRun{}
			unknownTaskRun.Status.Conditions = append(unknownTaskRun.Status.Conditions, apis.Condition{Type: apis.ConditionSucceeded, Reason: "random"})
			unknownBuildRun := v1alpha1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(&unknownBuildRun, &unknownTaskRun)
			Expect(unknownBuildRun.Status.Failure).To(BeNil())
		})
	})
})
