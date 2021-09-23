// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	"github.com/shipwright-io/build/test"
	"github.com/shipwright-io/build/test/utils"
)

var _ = Describe("GenerateTaskrun", func() {

	var (
		build                  *buildv1alpha1.Build
		buildRun               *buildv1alpha1.BuildRun
		buildStrategy          *buildv1alpha1.BuildStrategy
		builderImage           *buildv1alpha1.Image
		dockerfile, buildpacks string
		ctl                    test.Catalog
	)

	BeforeEach(func() {
		buildpacks = "buildpacks-v3"
		dockerfile = "Dockerfile"
	})

	Describe("Generate the TaskSpec", func() {
		var (
			expectedCommandOrArg []string
			got                  *v1beta1.TaskSpec
			err                  error
		)
		BeforeEach(func() {
			builderImage = &buildv1alpha1.Image{
				Image: "quay.io/builder/image",
			}
		})

		Context("when the task spec is generated", func() {
			BeforeEach(func() {
				build, err = ctl.LoadBuildYAML([]byte(test.BuildahBuildWithAnnotationAndLabel))
				Expect(err).To(BeNil())

				buildRun, err = ctl.LoadBuildRunFromBytes([]byte(test.MinimalBuildahBuildRun))
				Expect(err).To(BeNil())

				buildStrategy, err = ctl.LoadBuildStrategyFromBytes([]byte(test.MinimalBuildahBuildStrategy))
				Expect(err).To(BeNil())
				buildStrategy.Spec.BuildSteps[0].ImagePullPolicy = "Always"

				expectedCommandOrArg = []string{
					"bud", "--tag=$(params.shp-output-image)", fmt.Sprintf("--file=$(inputs.params.%s)", "DOCKERFILE"), "$(params.shp-source-context)",
				}
			})

			JustBeforeEach(func() {
				got, err = resources.GenerateTaskSpec(config.NewDefaultConfig(), build, buildRun, buildStrategy.Spec.BuildSteps, []buildv1alpha1.Parameter{})
				Expect(err).To(BeNil())
			})

			It("should contain a step to clone the Git sources", func() {
				Expect(got.Steps[0].Name).To(Equal("source-default"))
				Expect(got.Steps[0].Command[0]).To(Equal("/ko-app/git"))
				Expect(got.Steps[0].Args).To(Equal([]string{
					"--url",
					build.Spec.Source.URL,
					"--target",
					"$(params.shp-source-root)",
					"--result-file-commit-sha",
					"$(results.shp-source-default-commit-sha.path)",
					"--result-file-commit-author",
					"$(results.shp-source-default-commit-author.path)",
				}))
			})

			It("should contain results for the image", func() {
				Expect(got.Results).To(utils.ContainNamedElement("shp-image-digest"))
				Expect(got.Results).To(utils.ContainNamedElement("shp-image-size"))
			})

			It("should contain a result for the Git commit SHA", func() {
				Expect(got.Results).To(utils.ContainNamedElement("shp-source-default-commit-sha"))
			})

			It("should ensure IMAGE is replaced by builder image when needed.", func() {
				Expect(got.Steps[1].Container.Image).To(Equal("quay.io/containers/buildah:v1.20.1"))
			})

			It("should ensure ImagePullPolicy can be set by the build strategy author.", func() {
				Expect(got.Steps[1].Container.ImagePullPolicy).To(Equal(corev1.PullPolicy("Always")))
			})

			It("should ensure command replacements happen when needed", func() {
				Expect(got.Steps[1].Container.Command[0]).To(Equal("/usr/bin/buildah"))
			})

			It("should ensure resource replacements happen for the first step", func() {
				Expect(got.Steps[1].Container.Resources).To(Equal(ctl.LoadCustomResources("500m", "1Gi")))
			})

			It("should ensure resource replacements happen for the second step", func() {
				Expect(got.Steps[2].Container.Resources).To(Equal(ctl.LoadCustomResources("100m", "65Mi")))
			})

			It("should ensure arg replacements happen when needed", func() {
				Expect(got.Steps[1].Container.Args).To(Equal(expectedCommandOrArg))
			})

			It("should ensure top level volumes are populated", func() {
				Expect(len(got.Volumes)).To(Equal(1))
			})

			It("should contain the shipwright system parameters", func() {
				Expect(got.Params).To(utils.ContainNamedElement("shp-source-root"))
				Expect(got.Params).To(utils.ContainNamedElement("shp-source-context"))
				Expect(got.Params).To(utils.ContainNamedElement("shp-output-image"))

				// legacy params
				Expect(got.Params).ToNot(utils.ContainNamedElement("BUILDER_IMAGE")) // test build has no builder image
				Expect(got.Params).To(utils.ContainNamedElement("CONTEXT_DIR"))
				Expect(got.Params).To(utils.ContainNamedElement("DOCKERFILE"))

				Expect(len(got.Params)).To(Equal(5))
			})

			It("should contain a step to mutate the image with single mutate args", func() {
				Expect(got.Steps[3].Name).To(Equal("mutate-image"))
				Expect(got.Steps[3].Command[0]).To(Equal("/ko-app/mutate-image"))
				Expect(got.Steps[3].Args).To(Equal([]string{
					"--image",
					"$(params.shp-output-image)",
					"--result-file-image-digest",
					"$(results.shp-image-digest.path)",
					"result-file-image-size",
					"$(results.shp-image-size.path)",
					"--annotation",
					"org.opencontainers.image.url=https://my-company.com/images",
					"--label",
					"maintainer=team@my-company.com",
				}))
			})

			It("should contain a step to mutate the image with multiple mutate args", func() {
				build, err = ctl.LoadBuildYAML([]byte(test.BuildahBuildWithMultipleAnnotationAndLabel))
				Expect(err).To(BeNil())

				got, err = resources.GenerateTaskSpec(config.NewDefaultConfig(), build, buildRun, buildStrategy.Spec.BuildSteps, []buildv1alpha1.Parameter{})
				Expect(err).To(BeNil())

				Expect(got.Steps[3].Name).To(Equal("mutate-image"))
				Expect(got.Steps[3].Command[0]).To(Equal("/ko-app/mutate-image"))
				Expect(got.Steps[3].Args).Should(ConsistOf([]string{
					"--image",
					"$(params.shp-output-image)",
					"--result-file-image-digest",
					"$(results.shp-image-digest.path)",
					"result-file-image-size",
					"$(results.shp-image-size.path)",
					"--annotation",
					"org.opencontainers.image.source=https://github.com/org/repo",
					"--annotation",
					"org.opencontainers.image.url=https://my-company.com/images",
					"--label",
					"description=This is my cool image",
					"--label",
					"maintainer=team@my-company.com",
				}))
			})
		})
	})

	Describe("Generate the TaskRun", func() {
		var (
			k8sDuration30s                                                            *metav1.Duration
			k8sDuration1m                                                             *metav1.Duration
			namespace, contextDir, outputPath, outputPathBuildRun, serviceAccountName string
			got                                                                       *v1beta1.TaskRun
			err                                                                       error
		)
		BeforeEach(func() {
			duration, err := time.ParseDuration("30s")
			Expect(err).ToNot(HaveOccurred())
			k8sDuration30s = &metav1.Duration{
				Duration: duration,
			}
			duration, err = time.ParseDuration("1m")
			Expect(err).ToNot(HaveOccurred())
			k8sDuration1m = &metav1.Duration{
				Duration: duration,
			}

			namespace = "build-test"
			contextDir = "docker-build"
			builderImage = &buildv1alpha1.Image{
				Image: "heroku/buildpacks:18",
			}
			outputPath = "image-registry.openshift-image-registry.svc:5000/example/buildpacks-app"
			outputPathBuildRun = "image-registry.openshift-image-registry.svc:5000/example/buildpacks-app-v2"
			serviceAccountName = buildpacks + "-serviceaccount"
		})

		Context("when the taskrun is generated by default", func() {
			BeforeEach(func() {
				build, err = ctl.LoadBuildYAML([]byte(test.BuildahBuildWithOutput))
				Expect(err).To(BeNil())

				buildRun, err = ctl.LoadBuildRunFromBytes([]byte(test.BuildahBuildRunWithSA))
				Expect(err).To(BeNil())

				buildStrategy, err = ctl.LoadBuildStrategyFromBytes([]byte(test.BuildahBuildStrategySingleStep))
				Expect(err).To(BeNil())

			})

			JustBeforeEach(func() {
				got, err = resources.GenerateTaskRun(config.NewDefaultConfig(), build, buildRun, serviceAccountName, buildStrategy)
				Expect(err).To(BeNil())
			})

			It("should ensure generated TaskRun's basic information are correct", func() {
				Expect(strings.Contains(got.GenerateName, buildRun.Name+"-")).To(Equal(true))
				Expect(got.Namespace).To(Equal(namespace))
				Expect(got.Spec.ServiceAccountName).To(Equal(buildpacks + "-serviceaccount"))
				Expect(got.Labels[buildv1alpha1.LabelBuild]).To(Equal(build.Name))
				Expect(got.Labels[buildv1alpha1.LabelBuildRun]).To(Equal(buildRun.Name))
				Expect(got.Labels[buildv1alpha1.LabelBuildStrategyName]).To(Equal(build.Spec.Strategy.Name))
				Expect(got.Labels[buildv1alpha1.LabelBuildStrategyGeneration]).To(Equal("0"))
			})

			It("should filter out certain annotations when propagating them to the TaskRun", func() {
				Expect(len(got.Annotations)).To(Equal(2))
				Expect(got.Annotations["kubernetes.io/egress-bandwidth"]).To(Equal("1M"))
				Expect(got.Annotations["kubernetes.io/ingress-bandwidth"]).To(Equal("1M"))
			})

			It("should ensure generated TaskRun has no resources", func() {
				Expect(got.Spec.Resources).To(BeNil())
			})

			It("should ensure resource replacements happen when needed", func() {
				expectedResourceOrArg := corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				}
				Expect(got.Spec.TaskSpec.Steps[1].Resources).To(Equal(expectedResourceOrArg))
			})

			It("should have no timeout set", func() {
				Expect(got.Spec.Timeout).To(BeNil())
			})
		})

		Context("when the taskrun is generated by special settings", func() {
			BeforeEach(func() {
				build, err = ctl.LoadBuildYAML([]byte(test.BuildpacksBuildWithBuilderAndTimeOut))
				Expect(err).To(BeNil())

				buildRun, err = ctl.LoadBuildRunFromBytes([]byte(test.BuildpacksBuildRunWithSA))
				Expect(err).To(BeNil())

				buildStrategy, err = ctl.LoadBuildStrategyFromBytes([]byte(test.BuildpacksBuildStrategySingleStep))
				Expect(err).To(BeNil())
			})

			JustBeforeEach(func() {
				got, err = resources.GenerateTaskRun(config.NewDefaultConfig(), build, buildRun, serviceAccountName, buildStrategy)
				Expect(err).To(BeNil())
			})

			It("should ensure generated TaskRun's basic information are correct", func() {
				Expect(strings.Contains(got.GenerateName, buildRun.Name+"-")).To(Equal(true))
				Expect(got.Namespace).To(Equal(namespace))
				Expect(got.Spec.ServiceAccountName).To(Equal(buildpacks + "-serviceaccount"))
				Expect(got.Labels[buildv1alpha1.LabelBuild]).To(Equal(build.Name))
				Expect(got.Labels[buildv1alpha1.LabelBuildRun]).To(Equal(buildRun.Name))
			})

			It("should ensure generated TaskRun's spec special input params are correct", func() {
				params := got.Spec.Params

				paramSourceRootFound := false
				paramSourceContextFound := false
				paramOutputImageFound := false

				// legacy params
				paramBuilderImageFound := false
				paramDockerfileFound := false
				paramContextDirFound := false

				for _, param := range params {
					switch param.Name {
					case "shp-source-root":
						paramSourceRootFound = true
						Expect(param.Value.StringVal).To(Equal("/workspace/source"))

					case "shp-source-context":
						paramSourceContextFound = true
						Expect(param.Value.StringVal).To(Equal("/workspace/source/docker-build"))

					case "shp-output-image":
						paramOutputImageFound = true
						Expect(param.Value.StringVal).To(Equal(outputPath))

					case "BUILDER_IMAGE":
						paramBuilderImageFound = true
						Expect(param.Value.StringVal).To(Equal(builderImage.Image))

					case "DOCKERFILE":
						paramDockerfileFound = true
						Expect(param.Value.StringVal).To(Equal(dockerfile))

					case "CONTEXT_DIR":
						paramContextDirFound = true
						Expect(param.Value.StringVal).To(Equal(contextDir))

					default:
						Fail(fmt.Sprintf("Unexpected param found: %s", param.Name))
					}
				}

				Expect(paramSourceRootFound).To(BeTrue())
				Expect(paramSourceContextFound).To(BeTrue())
				Expect(paramOutputImageFound).To(BeTrue())

				Expect(paramBuilderImageFound).To(BeTrue())
				Expect(paramDockerfileFound).To(BeTrue())
				Expect(paramContextDirFound).To(BeTrue())
			})

			It("should ensure resource replacements happen when needed", func() {
				expectedResourceOrArg := corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				}
				Expect(got.Spec.TaskSpec.Steps[1].Resources).To(Equal(expectedResourceOrArg))
			})

			It("should have the timeout set correctly", func() {
				Expect(got.Spec.Timeout).To(Equal(k8sDuration30s))
			})
		})

		Context("when the build and buildrun contain a timeout", func() {
			BeforeEach(func() {
				build, err = ctl.LoadBuildYAML([]byte(test.BuildahBuildWithTimeOut))
				Expect(err).To(BeNil())

				buildRun, err = ctl.LoadBuildRunFromBytes([]byte(test.BuildahBuildRunWithTimeOutAndSA))
				Expect(err).To(BeNil())

				buildStrategy, err = ctl.LoadBuildStrategyFromBytes([]byte(test.BuildahBuildStrategySingleStep))
				Expect(err).To(BeNil())
			})

			JustBeforeEach(func() {
				got, err = resources.GenerateTaskRun(config.NewDefaultConfig(), build, buildRun, serviceAccountName, buildStrategy)
				Expect(err).To(BeNil())
			})

			It("should use the timeout from the BuildRun", func() {
				Expect(got.Spec.Timeout).To(Equal(k8sDuration1m))
			})
		})

		Context("when the build and buildrun both contain an output imageURL", func() {
			BeforeEach(func() {

				build, err = ctl.LoadBuildYAML([]byte(test.BuildahBuildWithOutput))
				Expect(err).To(BeNil())

				buildRun, err = ctl.LoadBuildRunFromBytes([]byte(test.BuildahBuildRunWithSAAndOutput))
				Expect(err).To(BeNil())

				buildStrategy, err = ctl.LoadBuildStrategyFromBytes([]byte(test.BuildahBuildStrategySingleStep))
				Expect(err).To(BeNil())
			})

			JustBeforeEach(func() {
				got, err = resources.GenerateTaskRun(config.NewDefaultConfig(), build, buildRun, serviceAccountName, buildStrategy)
				Expect(err).To(BeNil())
			})

			It("should use the imageURL from the BuildRun for the param", func() {
				params := got.Spec.Params

				paramOutputImageFound := false

				for _, param := range params {
					switch param.Name {
					case "shp-output-image":
						paramOutputImageFound = true
						Expect(param.Value.StringVal).To(Equal(outputPathBuildRun))
					}
				}

				Expect(paramOutputImageFound).To(BeTrue())
			})
		})
	})
})
