// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/test"
)

var _ = Describe("Integration tests Build and TaskRun", func() {
	var (
		cbsObject      *v1alpha1.ClusterBuildStrategy
		buildObject    *v1alpha1.Build
		buildRunObject *v1alpha1.BuildRun
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

				Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
				Expect(buildObject.Status.Reason).To(Equal(v1alpha1.SucceedStatus))
				Expect(buildObject.Status.Message).To(Equal(v1alpha1.AllValidationsSucceeded))
			})

			It("should be successful with label value as empty string", func() {
				buildObject.Spec.Output.Labels =
					map[string]string{
						"maintainer": "",
					}
				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
				Expect(buildObject.Status.Reason).To(Equal(v1alpha1.SucceedStatus))
				Expect(buildObject.Status.Message).To(Equal(v1alpha1.AllValidationsSucceeded))
			})

			It("should be successful with annotation", func() {
				buildObject.Spec.Output.Annotations =
					map[string]string{
						"org.opencontainers.image.url": "https://my-company.com/images",
					}
				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
				Expect(buildObject.Status.Reason).To(Equal(v1alpha1.SucceedStatus))
				Expect(buildObject.Status.Message).To(Equal(v1alpha1.AllValidationsSucceeded))
			})

			It("should be successful with label", func() {
				buildObject.Spec.Output.Labels =
					map[string]string{
						"maintainer": "team@my-company.com",
					}
				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
				Expect(buildObject.Status.Reason).To(Equal(v1alpha1.SucceedStatus))
				Expect(buildObject.Status.Message).To(Equal(v1alpha1.AllValidationsSucceeded))
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

				Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
				Expect(buildObject.Status.Reason).To(Equal(v1alpha1.SucceedStatus))
				Expect(buildObject.Status.Message).To(Equal(v1alpha1.AllValidationsSucceeded))
			})
		})

		Context("when creating the taskrun", func() {
			It("should contain a step to mutate the image", func() {
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

				Expect(tr.Spec.TaskSpec.Steps[3].Name).To(Equal("mutate-image"))
				Expect(tr.Spec.TaskSpec.Steps[3].Command[0]).To(Equal("/ko-app/mutate-image"))
				Expect(tr.Spec.TaskSpec.Steps[3].Args).To(Equal([]string{
					"--image",
					"$(params.shp-output-image)",
					"--result-file-image-digest",
					"$(results.shp-image-digest.path)",
					"result-file-image-size",
					"$(results.shp-image-size.path)",
					"--annotation",
					"org.opencontainers.image.url=https://my-company.com/images",
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

				for _, step := range tr.Spec.TaskSpec.Steps {
					Expect(step.Name).ToNot(Equal("mutate-image"))
				}
			})
		})
	})
})
