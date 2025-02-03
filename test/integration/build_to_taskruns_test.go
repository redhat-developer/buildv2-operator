// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	utils "github.com/shipwright-io/build/test/utils/v1beta1"
	test "github.com/shipwright-io/build/test/v1beta1_samples"
)

var _ = Describe("Integration tests Build and TaskRun", func() {
	var (
		cbsObject      *v1beta1.ClusterBuildStrategy
		buildObject    *v1beta1.Build
		buildRunObject *v1beta1.BuildRun
		buildSample,
		buildRunSample []byte
	)

	// Load the ClusterBuildStrategies before each test case
	BeforeEach(func() {
		cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategySingleStep))
		Expect(err).To(BeNil())

		err = tb.CreateClusterBuildStrategy(cbsObject)
		Expect(err).To(BeNil())
	})

	// Delete the ClusterBuildStrategies after each test case
	AfterEach(func() {
		_, err = tb.GetBuild(buildObject.Name)
		if err == nil {
			Expect(tb.DeleteBuild(buildObject.Name)).To(BeNil())
		}

		err = tb.DeleteClusterBuildStrategy(cbsObject.Name)
		Expect(err).To(BeNil())
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

	Context("when a build with annotation or label is defined", func() {
		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		Context("when creating the build", func() {
			It("should be successful with annotation value as empty string", func() {
				buildObject.Spec.Output.Annotations =
					map[string]string{
						"org.opencontainers.image.url": "",
					}
				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
				Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SucceedStatus))
				Expect(*buildObject.Status.Message).To(Equal(v1beta1.AllValidationsSucceeded))
			})

			It("should be successful with label value as empty string", func() {
				buildObject.Spec.Output.Labels =
					map[string]string{
						"maintainer": "",
					}
				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
				Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SucceedStatus))
				Expect(*buildObject.Status.Message).To(Equal(v1beta1.AllValidationsSucceeded))
			})

			It("should be successful with annotation", func() {
				buildObject.Spec.Output.Annotations =
					map[string]string{
						"org.opencontainers.image.url": "https://my-company.com/images",
					}
				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
				Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SucceedStatus))
				Expect(*buildObject.Status.Message).To(Equal(v1beta1.AllValidationsSucceeded))
			})

			It("should be successful with label", func() {
				buildObject.Spec.Output.Labels =
					map[string]string{
						"maintainer": "team@my-company.com",
					}
				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
				Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SucceedStatus))
				Expect(*buildObject.Status.Message).To(Equal(v1beta1.AllValidationsSucceeded))
			})

			It("should be successful with both label and annotation", func() {
				buildObject.Spec.Output.Annotations =
					map[string]string{
						"org.opencontainers.image.url": "https://my-company.com/images",
					}
				buildObject.Spec.Output.Labels =
					map[string]string{
						"maintainer": "team@my-company.com",
					}

				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
				Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SucceedStatus))
				Expect(*buildObject.Status.Message).To(Equal(v1beta1.AllValidationsSucceeded))
			})
		})

		Context("when creating the taskrun", func() {
			It("should contain an image-processing step to mutate the image", func() {
				buildObject.Spec.Output.Annotations =
					map[string]string{
						"org.opencontainers.image.url": "https://my-company.com/images",
					}
				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				Expect(tb.CreateBR(buildRunObject)).To(BeNil())

				_, err = tb.GetBRTillStartTime(buildRunObject.Name)
				Expect(err).To(BeNil())

				tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
				Expect(err).To(BeNil())

				Expect(tr.Spec.TaskSpec.Steps[3].Name).To(Equal("image-processing"))
				Expect(tr.Spec.TaskSpec.Steps[3].Command[0]).To(Equal("/ko-app/image-processing"))
				Expect(tr.Spec.TaskSpec.Steps[3].Args).To(Equal([]string{
					"--annotation",
					"org.opencontainers.image.url=https://my-company.com/images",
					"--image",
					"$(params.shp-output-image)",
					"--insecure=$(params.shp-output-insecure)",
					"--result-file-image-digest",
					"$(results.shp-image-digest.path)",
					"--result-file-image-size",
					"$(results.shp-image-size.path)",
					"--result-file-image-vulnerabilities",
					"$(results.shp-image-vulnerabilities.path)",
				}))
			})

			It("should not contain a step incase annotation or label are not specified", func() {
				buildObject.Spec.Output.Annotations = nil
				buildObject.Spec.Output.Labels = nil

				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				Expect(tb.CreateBR(buildRunObject)).To(BeNil())

				_, err = tb.GetBRTillStartTime(buildRunObject.Name)
				Expect(err).To(BeNil())

				tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
				Expect(err).To(BeNil())

				Expect(tr.Spec.TaskSpec.Steps).ToNot(utils.ContainNamedElement("image-processing"))
			})
		})
	})

	Context("when a build with nodeSelector is defined", func() {
		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithNodeSelector)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		Context("when the TaskRun is created", func() {
			It("should have the nodeSelector specified in the PodTemplate", func() {
				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())
				Expect(*buildObject.Status.Message).To(Equal(v1beta1.AllValidationsSucceeded))
				Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
				Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SucceedStatus))

				Expect(tb.CreateBR(buildRunObject)).To(BeNil())

				_, err = tb.GetBRTillStartTime(buildRunObject.Name)
				Expect(err).To(BeNil())

				tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
				Expect(err).To(BeNil())
				Expect(buildObject.Spec.NodeSelector).To(Equal(tr.Spec.PodTemplate.NodeSelector))
			})
		})

		Context("when the nodeSelector is invalid", func() {
			It("fails the build with a proper error in Reason", func() {
				// set nodeSelector label to be invalid
				buildObject.Spec.NodeSelector = map[string]string{strings.Repeat("s", 64): ""}
				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
				Expect(*buildObject.Status.Reason).To(Equal(v1beta1.NodeSelectorNotValid))
			})
		})
	})

	Context("when a build with Tolerations is defined", func() {
		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithToleration)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		Context("when the TaskRun is created", func() {
			It("should have the Tolerations specified in the PodTemplate", func() {
				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())
				Expect(*buildObject.Status.Message).To(Equal(v1beta1.AllValidationsSucceeded))
				Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
				Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SucceedStatus))

				Expect(tb.CreateBR(buildRunObject)).To(BeNil())

				_, err = tb.GetBRTillStartTime(buildRunObject.Name)
				Expect(err).To(BeNil())

				tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
				Expect(err).To(BeNil())
				Expect(buildObject.Spec.Tolerations[0].Key).To(Equal(tr.Spec.PodTemplate.Tolerations[0].Key))
				Expect(buildObject.Spec.Tolerations[0].Operator).To(Equal(tr.Spec.PodTemplate.Tolerations[0].Operator))
				Expect(buildObject.Spec.Tolerations[0].Value).To(Equal(tr.Spec.PodTemplate.Tolerations[0].Value))
				Expect(tr.Spec.PodTemplate.Tolerations[0].TolerationSeconds).To(Equal(corev1.Toleration{}.TolerationSeconds))
				Expect(tr.Spec.PodTemplate.Tolerations[0].Effect).To(Equal(corev1.TaintEffectNoSchedule))
			})
		})

		Context("when the Toleration is invalid", func() {
			It("fails the build with a proper error in Reason", func() {
				// set Toleration Key to be invalid
				buildObject.Spec.Tolerations[0].Key = strings.Repeat("s", 64)
				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
				Expect(*buildObject.Status.Reason).To(Equal(v1beta1.TolerationNotValid))
			})
		})
	})
})
