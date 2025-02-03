// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"

	"github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	test "github.com/shipwright-io/build/test/v1beta1_samples"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
)

var _ = Describe("Integration tests BuildRuns and TaskRuns", func() {
	var (
		cbsObject      *v1beta1.ClusterBuildStrategy
		buildObject    *v1beta1.Build
		buildRunObject *v1beta1.BuildRun
		buildSample    []byte
		buildRunSample []byte
	)

	var setupBuildAndBuildRun = func(buildDef []byte, buildRunDef []byte, strategy ...string) (watch.Interface, *v1beta1.Build, *v1beta1.BuildRun) {

		var strategyName = STRATEGY + tb.Namespace
		if len(strategy) > 0 {
			strategyName = strategy[0]
		}

		timeout := int64(tb.TimeOut.Seconds())
		buildRunWatcher, err := tb.BuildClientSet.ShipwrightV1beta1().BuildRuns(tb.Namespace).Watch(tb.Context, metav1.ListOptions{TimeoutSeconds: &timeout})
		Expect(err).To(BeNil())

		buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, strategyName, buildDef)
		Expect(err).To(BeNil())
		Expect(tb.CreateBuild(buildObject)).To(BeNil())

		buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
		Expect(err).To(BeNil())

		buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, BUILD+tb.Namespace, buildRunDef)
		Expect(err).To(BeNil())
		Expect(tb.CreateBR(buildRunObject)).To(BeNil())

		//TODO: consider how to deal with buildObject or buildRunObject
		return buildRunWatcher, buildObject, buildRunObject
	}

	var WithCustomClusterBuildStrategy = func(data []byte, f func()) {
		customClusterBuildStrategy, err := tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace+"custom", data)
		Expect(err).To(BeNil())

		Expect(tb.CreateClusterBuildStrategy(customClusterBuildStrategy)).To(BeNil())
		f()
		Expect(tb.DeleteClusterBuildStrategy(customClusterBuildStrategy.Name)).To(BeNil())
	}

	// Load the ClusterBuildStrategies before each test case
	BeforeEach(func() {
		cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategySingleStepKaniko))
		Expect(err).To(BeNil())

		err = tb.CreateClusterBuildStrategy(cbsObject)
		Expect(err).To(BeNil())
	})

	// Delete the ClusterBuildStrategies after each test case
	AfterEach(func() {
		if buildObject != nil {
			_, err = tb.GetBuild(buildObject.Name)
			if err == nil {
				Expect(tb.DeleteBuild(buildObject.Name)).To(BeNil())
			}
			buildObject = nil
		}

		if cbsObject != nil {
			err := tb.DeleteClusterBuildStrategy(cbsObject.Name)
			Expect(err).To(BeNil())
			cbsObject = nil
		}
	})

	// Override the Builds and BuildRuns CRDs instances to use
	// before an It() statement is executed
	JustBeforeEach(func() {
		if buildSample != nil {
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, STRATEGY+tb.Namespace, buildSample)
			Expect(err).To(BeNil())
		}

		if buildRunSample != nil {
			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, BUILD+tb.Namespace, buildRunSample)
			Expect(err).To(BeNil())
		}
	})

	Context("when buildrun uses conditions", func() {
		Context("when condition status unknown", func() {
			It("reflects a change from pending to running reason", func() {
				buildRunWatcher, _, _ := setupBuildAndBuildRun([]byte(test.BuildCBSMinimal), []byte(test.MinimalBuildRun))

				// use a fakeTime to simplify tests
				fakeTime := time.Date(1989, 05, 15, 00, 01, 01, 651387237, time.UTC)

				var seq = []*v1beta1.Condition{}
				for event := range buildRunWatcher.ResultChan() {
					if event.Type == watch.Error {
						GinkgoWriter.Write([]byte(fmt.Sprintf("Unexpected error event in watch: %v", event.Object)))
						continue
					}

					condition := event.Object.(*v1beta1.BuildRun).Status.GetCondition(v1beta1.Succeeded)
					if condition != nil {
						condition.LastTransitionTime = metav1.Time{Time: fakeTime}
						seq = append(seq, condition)
					}

					// Pending -> Running
					if condition != nil && condition.Reason == "Running" {
						buildRunWatcher.Stop()
					}
				}
				// consider a longer sequence, for events where the cluster have
				// insufficient resources, where the Reason will be ExceededNodeResources.
				Expect(len(seq) >= 2).To(Equal(true))

				// ensure the conditions move eventually from unknown into running
				Expect(seq).Should(ContainElement(&v1beta1.Condition{
					Type:               v1beta1.Succeeded,
					Status:             corev1.ConditionUnknown,
					LastTransitionTime: metav1.Time{Time: fakeTime},
					Reason:             "Pending",
					Message:            "Pending",
				}))
				Expect(seq).Should(ContainElement(&v1beta1.Condition{
					Type:               v1beta1.Succeeded,
					Status:             corev1.ConditionUnknown,
					LastTransitionTime: metav1.Time{Time: fakeTime},
					Reason:             "Running",
					Message:            "Not all Steps in the Task have finished executing",
				}))
			})
		})

		Context("when condition status is false", func() {
			It("reflects a timeout", func() {
				_, build, buildRun := setupBuildAndBuildRun([]byte(test.BuildCBSWithShortTimeOut), []byte(test.MinimalBuildRun))

				buildRun, err := tb.GetBRTillCompletion(buildRun.Name)
				Expect(err).ToNot(HaveOccurred())

				condition := buildRun.Status.GetCondition(v1beta1.Succeeded)
				Expect(condition.Status).To(Equal(corev1.ConditionFalse))
				Expect(condition.Reason).To(Equal("BuildRunTimeout"))
				Expect(condition.Message).To(Equal(fmt.Sprintf("BuildRun %s failed to finish within %v", buildRun.Name, build.Spec.Timeout.Duration)))
			})

			It("reflects a failed reason", func() {
				WithCustomClusterBuildStrategy([]byte(test.ClusterBuildStrategySingleStepKanikoError), func() {
					_, _, buildRun := setupBuildAndBuildRun([]byte(test.BuildCBSMinimal), []byte(test.MinimalBuildRun), STRATEGY+tb.Namespace+"custom")

					buildRun, err := tb.GetBRTillCompletion(buildRun.Name)
					Expect(err).ToNot(HaveOccurred())
					Expect(buildRun.Status.CompletionTime).ToNot(BeNil())

					taskRun, err := tb.GetTaskRunFromBuildRun(buildRun.Name)
					Expect(err).ToNot(HaveOccurred())

					Expect(buildRun.Status.FailureDetails.Location.Pod).To(Equal(taskRun.Status.PodName))
					Expect(buildRun.Status.FailureDetails.Location.Container).To(Equal("step-step-build-and-push"))

					condition := buildRun.Status.GetCondition(v1beta1.Succeeded)
					Expect(condition.Status).To(Equal(corev1.ConditionFalse))
					Expect(condition.Reason).To(Equal("Failed"))
					Expect(condition.Message).To(ContainSubstring("buildrun step %s failed", "step-step-build-and-push"))
				})
			})
		})

		Context("when condition status true", func() {
			It("should reflect the taskrun succeeded reason in the buildrun condition", func() {
				WithCustomClusterBuildStrategy([]byte(test.ClusterBuildStrategyNoOp), func() {
					_, _, buildRun := setupBuildAndBuildRun([]byte(test.BuildCBSMinimal), []byte(test.MinimalBuildRun), STRATEGY+tb.Namespace+"custom")

					buildRun, err := tb.GetBRTillCompletion(buildRun.Name)
					Expect(err).ToNot(HaveOccurred())

					condition := buildRun.Status.GetCondition(v1beta1.Succeeded)
					Expect(condition.Status).To(Equal(corev1.ConditionTrue))
					Expect(condition.Reason).To(Equal("Succeeded"))
					Expect(condition.Message).To(ContainSubstring("All Steps have completed executing"))
				})
			})
		})
	})

	Context("when a buildrun is created", func() {
		It("should reflect a Pending and Running reason", func() {
			// use a custom strategy here that just sleeps 30 seconds to prevent it from completing too fast so that we do not get the Running state
			WithCustomClusterBuildStrategy([]byte(test.ClusterBuildStrategySleep30s), func() {
				_, _, buildRunObject := setupBuildAndBuildRun([]byte(test.BuildCBSMinimal), []byte(test.MinimalBuildRun), STRATEGY+tb.Namespace+"custom")

				_, err = tb.GetBRTillStartTime(buildRunObject.Name)
				Expect(err).To(BeNil())

				// Pending is an intermediate state where a certain amount of luck is needed to capture it with a polling interval of 3s.
				// Also, if the build-operator is not reconciling on this TaskRun status quick enough, a BuildRun might never be in Pending
				// but rather directly go to Running.
				/*
					expectedReason := "Pending"
					actualReason, err := tb.GetTRTillDesiredReason(buildRunObject.Name, expectedReason)
					Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))

					expectedReason = "Pending"
					actualReason, err = tb.GetBRTillDesiredReason(buildRunObject.Name, expectedReason)
					Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))
				*/

				expectedReason := "Running"
				actualReason, err := tb.GetTRTillDesiredReason(buildRunObject.Name, expectedReason)
				Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))

				expectedReason = "Running"
				actualReason, err = tb.GetBRTillDesiredReason(buildRunObject.Name, expectedReason)
				Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))
			})
		})
	})

	Context("when a buildrun is created but fails because of a timeout", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSWithShortTimeOut)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should reflect a TaskRunTimeout reason and Completion time on timeout", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())

			expectedReason := "TaskRunTimeout"
			actualReason, err := tb.GetTRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))

			_, err = tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			expectedReason = "BuildRunTimeout"
			actualReason, err = tb.GetBRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))

			tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(tr.Status.CompletionTime).ToNot(BeNil())
		})
	})

	Context("when a buildrun is created with a wrong url", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSWithWrongURL)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should reflect a Failed reason and Completion on failure", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			b, err := tb.GetBuildTillRegistration(buildObject.Name, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			Expect(*b.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(*b.Status.Reason).To(Equal(v1beta1.RemoteRepositoryUnreachable))
			Expect(*b.Status.Message).To(ContainSubstring("no such host"))

			_, err = tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())

			reason, err := tb.GetBRReason(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(reason).To(Equal(resources.ConditionBuildRegistrationFailed))

			_, err = tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).ToNot(BeNil())

		})
	})

	Context("when a buildrun is created and the taskrun is cancelled", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should reflect a TaskRunCancelled reason and no completionTime", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			tr.Spec.Status = "TaskRunCancelled"

			_, err = tb.UpdateTaskRun(tr.Name, func(tr *pipelineapi.TaskRun) {
				tr.Spec.Status = "TaskRunCancelled"
			})
			Expect(err).To(BeNil())

			expectedReason := "TaskRunCancelled"
			actualReason, err := tb.GetTRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))
		})
	})

	Context("when a standalone buildrun is created and the buildrun is cancelled", func() {

		var standAloneBuildRunSample []byte
		var standaloneBuildRunObject *v1beta1.BuildRun

		BeforeEach(func() {
			standAloneBuildRunSample = []byte(test.MinimalOneOffBuildRun)

			standaloneBuildRunObject, err = tb.Catalog.LoadStandAloneBuildRunWithNameAndStrategy(BUILDRUN+tb.Namespace+"-standalone", cbsObject, standAloneBuildRunSample)
			Expect(err).To(BeNil())
		})

		It("should reflect a TaskRunCancelled reason and no completionTime", func() {
			Expect(tb.CreateBR(standaloneBuildRunObject)).ToNot(HaveOccurred())

			br, err := tb.GetBRTillStartTime(standaloneBuildRunObject.Name)
			Expect(err).ToNot(HaveOccurred())

			_, err = tb.GetTaskRunFromBuildRun(br.Name)
			Expect(err).ToNot(HaveOccurred())

			br.Spec.State = v1beta1.BuildRunRequestedStatePtr(v1beta1.BuildRunStateCancel)
			data := []byte(fmt.Sprintf(`{"spec":{"state": "%s"}}`, v1beta1.BuildRunStateCancel))
			br, err = tb.BuildClientSet.ShipwrightV1beta1().
				BuildRuns(tb.Namespace).
				Patch(tb.Context, br.Name, types.MergePatchType, data, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			actualReason, err := tb.GetTRTillDesiredReason(br.Name, "TaskRunCancelled")
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", "TaskRunCancelled", actualReason))

			actualReason, err = tb.GetBRTillDesiredReason(br.Name, v1beta1.BuildRunStateCancel)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get desired BuildRun reason; expected %s, got %s", v1beta1.BuildRunStateCancel, actualReason))
			Expect(actualReason).To(Equal(v1beta1.BuildRunStateCancel))
		})
	})

	Context("when a buildrun is created and the buildrun is cancelled", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should reflect a TaskRunCancelled reason in the taskrun, BuildRunCanceled in the buildrun, and no completionTime", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			err := wait.PollUntilContextTimeout(tb.Context, 1*time.Second, 4*time.Second, true, func(_ context.Context) (done bool, err error) {
				bro, err := tb.GetBRTillStartTime(buildRunObject.Name)
				if err != nil {
					GinkgoT().Logf("error on br get: %s\n", err.Error())
					return false, nil
				}

				bro.Spec.State = v1beta1.BuildRunRequestedStatePtr(v1beta1.BuildRunStateCancel)
				err = tb.UpdateBR(bro)
				if err != nil {
					GinkgoT().Logf("error on br update: %s\n", err.Error())
					return false, nil
				}
				return true, nil
			})
			Expect(err).To(BeNil())

			expectedReason := "TaskRunCancelled"
			actualReason, err := tb.GetTRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired TaskRun reason; expected %s, got %s", expectedReason, actualReason))

			expectedReason = v1beta1.BuildRunStateCancel
			actualReason, err = tb.GetBRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired BuildRun reason; expected %s, got %s", expectedReason, actualReason))
		})
	})

	Context("when a buildrun is created and the taskrun deleted before completion", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should reflect a Failed reason", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			tb.DeleteTR(tr.Name)

			expectedReason := "TaskRunIsMissing"
			actualReason, err := tb.GetBRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))
		})
	})

	Context("when a buildrun is created and the taskrun deleted after successful completion", func() {

		It("should reflect a Success reason", func() {
			WithCustomClusterBuildStrategy([]byte(test.ClusterBuildStrategyNoOp), func() {
				_, _, buildRunObject := setupBuildAndBuildRun([]byte(test.BuildCBSMinimal), []byte(test.MinimalBuildRun), STRATEGY+tb.Namespace+"custom")

				_, err = tb.GetBRTillCompletion(buildRunObject.Name)
				Expect(err).To(BeNil())

				reason, err := tb.GetBRReason(buildRunObject.Name)
				Expect(err).To(BeNil())
				Expect(reason).To(Equal("Succeeded"))

				tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
				Expect(err).To(BeNil())
				Expect(tr.Status.CompletionTime).NotTo(BeNil())

				tb.DeleteTR(tr.Name)

				// in a test case, it is hard to verify that something (marking the BuildRun failed) is not happening, we quickly check the TaskRun is gone and
				// check one more time that the BuildRun is still Succeeded
				_, err = tb.GetTaskRunFromBuildRun(buildRunObject.Name)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("failed to find an owned TaskRun"))

				reason, err = tb.GetBRReason(buildRunObject.Name)
				Expect(err).To(BeNil())
				Expect(reason).To(Equal("Succeeded"))
			})
		})
	})

	Context("when a buildrun is created with invalid name", func() {
		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("fails the buildrun with a proper error in Reason", func() {
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Set buildrun name more than 63 characters
			buildRunObject.Name = strings.Repeat("s", 64)
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())

			condition := br.Status.GetCondition(v1beta1.Succeeded)
			Expect(condition.Status).To(Equal(corev1.ConditionFalse))
			Expect(condition.Reason).To(Equal(resources.BuildRunNameInvalid))
			Expect(condition.Message).To(Equal("must be no more than 63 characters"))
		})

		It("should reflect a BadRequest reason in TaskRun", func() {
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Set buildrun name more than 63 characters
			buildRunObject.Name = strings.Repeat("s", 64)
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())

			expectedReason := "BadRequest"
			actualReason, err := tb.GetTRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(HaveOccurred(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))

			_, err = tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when a buildrun is created with a nodeSelector defined", func() {
		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuild)
			buildRunSample = []byte(test.MinimalBuildRunWithNodeSelector)
		})

		Context("when the taskrun is created", func() {
			It("should have the nodeSelector specified in the PodTemplate", func() {
				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				Expect(tb.CreateBR(buildRunObject)).To(BeNil())

				br, err := tb.GetBRTillCompletion(buildRunObject.Name)
				Expect(err).To(BeNil())

				tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
				Expect(err).To(BeNil())
				Expect(br.Spec.NodeSelector).To(Equal(tr.Spec.PodTemplate.NodeSelector))
			})
		})

		Context("when the nodeSelector is invalid", func() {
			It("fails the buildrun with a proper error in Reason", func() {
				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				// set nodeSelector label to be invalid
				buildRunObject.Spec.NodeSelector = map[string]string{strings.Repeat("s", 64): "amd64"}
				Expect(tb.CreateBR(buildRunObject)).To(BeNil())

				br, err := tb.GetBRTillCompletion(buildRunObject.Name)
				Expect(err).To(BeNil())

				condition := br.Status.GetCondition(v1beta1.Succeeded)
				Expect(condition.Status).To(Equal(corev1.ConditionFalse))
				Expect(condition.Reason).To(Equal("PodCreationFailed"))
				Expect(condition.Message).To(ContainSubstring("must be no more than 63 characters"))
			})
		})
	})

	Context("when a buildrun is created with a Toleration defined", func() {
		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuild)
			buildRunSample = []byte(test.MinimalBuildRunWithToleration)
		})

		Context("when the taskrun is created", func() {
			It("should have the Toleration specified in the PodTemplate", func() {
				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				Expect(tb.CreateBR(buildRunObject)).To(BeNil())

				br, err := tb.GetBRTillCompletion(buildRunObject.Name)
				Expect(err).To(BeNil())

				tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
				Expect(err).To(BeNil())
				Expect(br.Spec.Tolerations[0].Key).To(Equal(tr.Spec.PodTemplate.Tolerations[0].Key))
				Expect(br.Spec.Tolerations[0].Operator).To(Equal(tr.Spec.PodTemplate.Tolerations[0].Operator))
				Expect(br.Spec.Tolerations[0].Value).To(Equal(tr.Spec.PodTemplate.Tolerations[0].Value))
				Expect(tr.Spec.PodTemplate.Tolerations[0].TolerationSeconds).To(Equal(corev1.Toleration{}.TolerationSeconds))
				Expect(tr.Spec.PodTemplate.Tolerations[0].Effect).To(Equal(corev1.TaintEffectNoSchedule))
			})
		})

		Context("when the Toleration is invalid", func() {
			It("fails the buildrun with a proper error in Reason", func() {
				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				// set Toleration Key to be invalid
				buildRunObject.Spec.Tolerations[0].Key = strings.Repeat("s", 64)
				Expect(tb.CreateBR(buildRunObject)).To(BeNil())

				br, err := tb.GetBRTillCompletion(buildRunObject.Name)
				Expect(err).To(BeNil())

				condition := br.Status.GetCondition(v1beta1.Succeeded)
				Expect(condition.Status).To(Equal(corev1.ConditionFalse))
				Expect(condition.Reason).To(Equal("PodCreationFailed"))
				Expect(condition.Message).To(ContainSubstring(validation.MaxLenError(63)))
			})
		})
	})
})
