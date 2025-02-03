// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/env"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	utils "github.com/shipwright-io/build/test/utils/v1beta1"
	test "github.com/shipwright-io/build/test/v1beta1_samples"
)

var _ = Describe("GenerateTaskrun", func() {
	var (
		build                  *buildv1beta1.Build
		buildWithEnvs          *buildv1beta1.Build
		buildRun               *buildv1beta1.BuildRun
		buildRunWithEnvs       *buildv1beta1.BuildRun
		buildStrategy          *buildv1beta1.BuildStrategy
		buildStrategyStepNames map[string]struct{}
		buildStrategyWithEnvs  *buildv1beta1.BuildStrategy
		buildpacks             string
		ctl                    test.Catalog
	)

	BeforeEach(func() {
		buildpacks = "buildpacks-v3"
	})

	Describe("Generate the TaskSpec", func() {
		var (
			expectedCommandOrArg []string
			got                  *pipelineapi.TaskSpec
			err                  error
		)

		Context("when the task spec is generated", func() {
			BeforeEach(func() {
				build, err = ctl.LoadBuildYAML([]byte(test.BuildahBuildWithAnnotationAndLabel))
				Expect(err).To(BeNil())

				buildRun, err = ctl.LoadBuildRunFromBytes([]byte(test.MinimalBuildahBuildRun))
				Expect(err).To(BeNil())

				buildStrategy, err = ctl.LoadBuildStrategyFromBytes([]byte(test.MinimalBuildahBuildStrategy))
				Expect(err).To(BeNil())
				buildStrategy.Spec.Steps[0].ImagePullPolicy = "Always"

				expectedCommandOrArg = []string{
					"bud", "--tag=$(params.shp-output-image)", "--file=$(params.dockerfile)", "$(params.shp-source-context)",
				}
			})

			JustBeforeEach(func() {
				taskRun, err := resources.GenerateTaskRun(config.NewDefaultConfig(), build, buildRun, "", buildStrategy)
				Expect(err).ToNot(HaveOccurred())
				got = taskRun.Spec.TaskSpec
			})

			It("should contain a step to clone the Git sources", func() {
				Expect(got.Steps[0].Name).To(Equal("source-default"))
				Expect(got.Steps[0].Command[0]).To(Equal("/ko-app/git"))
				Expect(got.Steps[0].Args).To(Equal([]string{
					"--url", build.Spec.Source.Git.URL,
					"--target", "$(params.shp-source-root)",
					"--result-file-commit-sha", "$(results.shp-source-default-commit-sha.path)",
					"--result-file-commit-author", "$(results.shp-source-default-commit-author.path)",
					"--result-file-branch-name", "$(results.shp-source-default-branch-name.path)",
					"--result-file-error-message", "$(results.shp-error-message.path)",
					"--result-file-error-reason", "$(results.shp-error-reason.path)",
					"--result-file-source-timestamp", "$(results.shp-source-default-source-timestamp.path)",
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
				Expect(got.Steps[1].Image).To(Equal("quay.io/containers/buildah:v1.38.1"))
			})

			It("should ensure ImagePullPolicy can be set by the build strategy author.", func() {
				Expect(got.Steps[1].ImagePullPolicy).To(Equal(corev1.PullPolicy("Always")))
			})

			It("should ensure command replacements happen when needed", func() {
				Expect(got.Steps[1].Command[0]).To(Equal("/usr/bin/buildah"))
			})

			It("should ensure resource replacements happen for the first step", func() {
				Expect(got.Steps[1].ComputeResources).To(Equal(ctl.LoadCustomResources("500m", "1Gi")))
			})

			It("should ensure resource replacements happen for the second step", func() {
				Expect(got.Steps[2].ComputeResources).To(Equal(ctl.LoadCustomResources("100m", "65Mi")))
			})

			It("should ensure arg replacements happen when needed", func() {
				Expect(got.Steps[1].Args).To(Equal(expectedCommandOrArg))
			})

			It("should ensure top level volumes are populated", func() {
				Expect(len(got.Volumes)).To(Equal(1))
			})

			It("should contain the shipwright system parameters", func() {
				Expect(got.Params).To(utils.ContainNamedElement("shp-source-root"))
				Expect(got.Params).To(utils.ContainNamedElement("shp-source-context"))
				Expect(got.Params).To(utils.ContainNamedElement("shp-output-image"))
				Expect(got.Params).To(utils.ContainNamedElement("shp-output-insecure"))

				// legacy params have been removed
				Expect(got.Params).ToNot(utils.ContainNamedElement("BUILDER_IMAGE"))
				Expect(got.Params).ToNot(utils.ContainNamedElement("CONTEXT_DIR"))

				Expect(len(got.Params)).To(Equal(5))
			})

			It("should contain a step to mutate the image with single mutate args", func() {
				Expect(got.Steps).To(HaveLen(4))
				Expect(got.Steps[3].Name).To(Equal("image-processing"))
				Expect(got.Steps[3].Command[0]).To(Equal("/ko-app/image-processing"))
				Expect(got.Steps[3].Args).To(BeEquivalentTo([]string{
					"--annotation",
					"org.opencontainers.image.url=https://my-company.com/images",
					"--label",
					"maintainer=team@my-company.com",
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

			It("should contain a step to mutate the image with multiple mutate args", func() {
				build, err = ctl.LoadBuildYAML([]byte(test.BuildahBuildWithMultipleAnnotationAndLabel))
				Expect(err).To(BeNil())

				taskRun, err := resources.GenerateTaskRun(config.NewDefaultConfig(), build, buildRun, "", buildStrategy)
				Expect(err).ToNot(HaveOccurred())
				got = taskRun.Spec.TaskSpec

				Expect(got.Steps[3].Name).To(Equal("image-processing"))
				Expect(got.Steps[3].Command[0]).To(Equal("/ko-app/image-processing"))

				expected := []string{
					"--annotation",
					"org.opencontainers.image.source=https://github.com/org/repo",
					"--annotation",
					"org.opencontainers.image.url=https://my-company.com/images",
					"--label",
					"description=This is my cool image",
					"--label",
					"maintainer=team@my-company.com",
					"--image",
					"$(params.shp-output-image)",
					"--insecure=$(params.shp-output-insecure)",
					"--result-file-image-digest",
					"$(results.shp-image-digest.path)",
					"--result-file-image-size",
					"$(results.shp-image-size.path)",
					"--result-file-image-vulnerabilities",
					"$(results.shp-image-vulnerabilities.path)",
				}

				Expect(got.Steps[3].Args).To(HaveLen(len(expected)))

				// there is no way to say which annotation comes first

				Expect(got.Steps[3].Args[1]).To(BeElementOf(expected[1], expected[3]))
				Expect(got.Steps[3].Args[3]).To(BeElementOf(expected[1], expected[3]))
				Expect(got.Steps[3].Args[1]).ToNot(Equal(got.Steps[3].Args[3]))

				expected[1] = got.Steps[3].Args[1]
				expected[3] = got.Steps[3].Args[3]

				// same for labels

				Expect(got.Steps[3].Args[5]).To(BeElementOf(expected[5], expected[7]))
				Expect(got.Steps[3].Args[7]).To(BeElementOf(expected[5], expected[7]))
				Expect(got.Steps[3].Args[5]).ToNot(Equal(got.Steps[3].Args[7]))

				expected[5] = got.Steps[3].Args[5]
				expected[7] = got.Steps[3].Args[7]

				Expect(got.Steps[3].Args).To(BeEquivalentTo(expected))
			})
		})

		Context("when env vars are defined", func() {
			BeforeEach(func() {
				build, err = ctl.LoadBuildYAML([]byte(test.MinimalBuildahBuild))
				Expect(err).To(BeNil())

				buildWithEnvs, err = ctl.LoadBuildYAML([]byte(test.MinimalBuildahBuildWithEnvVars))
				Expect(err).To(BeNil())

				buildRun, err = ctl.LoadBuildRunFromBytes([]byte(test.MinimalBuildahBuildRun))
				Expect(err).To(BeNil())

				buildRunWithEnvs, err = ctl.LoadBuildRunFromBytes([]byte(test.MinimalBuildahBuildRunWithEnvVars))
				Expect(err).To(BeNil())

				buildStrategy, err = ctl.LoadBuildStrategyFromBytes([]byte(test.MinimalBuildahBuildStrategy))
				Expect(err).To(BeNil())
				buildStrategy.Spec.Steps[0].ImagePullPolicy = "Always"
				buildStrategyStepNames = make(map[string]struct{})
				for _, step := range buildStrategy.Spec.Steps {
					buildStrategyStepNames[step.Name] = struct{}{}
				}

				buildStrategyWithEnvs, err = ctl.LoadBuildStrategyFromBytes([]byte(test.MinimalBuildahBuildStrategyWithEnvs))
				Expect(err).To(BeNil())

				expectedCommandOrArg = []string{
					"--storage-driver=$(params.storage-driver)", "bud", "--tag=$(params.shp-output-image)", fmt.Sprintf("--file=$(inputs.params.%s)", "DOCKERFILE"), "$(params.shp-source-context)",
				}
			})

			It("should contain env vars specified in Build in every BuildStrategy step", func() {
				got, err = resources.GenerateTaskSpec(config.NewDefaultConfig(), buildWithEnvs, buildRun, buildStrategy.Spec.Steps, []buildv1beta1.Parameter{}, buildStrategy.GetVolumes())
				Expect(err).To(BeNil())

				combinedEnvs, err := env.MergeEnvVars(buildRun.Spec.Env, buildWithEnvs.Spec.Env, true)
				Expect(err).NotTo(HaveOccurred())

				for _, step := range got.Steps {
					if _, ok := buildStrategyStepNames[step.Name]; ok {
						Expect(len(step.Env)).To(Equal(len(combinedEnvs)))
						Expect(reflect.DeepEqual(combinedEnvs, step.Env)).To(BeTrue())
					} else {
						Expect(reflect.DeepEqual(combinedEnvs, step.Env)).To(BeFalse())
					}
				}
			})

			It("should contain env vars specified in BuildRun in every step", func() {
				got, err = resources.GenerateTaskSpec(config.NewDefaultConfig(), build, buildRunWithEnvs, buildStrategy.Spec.Steps, []buildv1beta1.Parameter{}, buildStrategy.GetVolumes())
				Expect(err).To(BeNil())

				combinedEnvs, err := env.MergeEnvVars(buildRunWithEnvs.Spec.Env, build.Spec.Env, true)
				Expect(err).NotTo(HaveOccurred())

				for _, step := range got.Steps {
					if _, ok := buildStrategyStepNames[step.Name]; ok {
						Expect(len(step.Env)).To(Equal(len(combinedEnvs)))
						Expect(reflect.DeepEqual(combinedEnvs, step.Env)).To(BeTrue())
					} else {
						Expect(reflect.DeepEqual(combinedEnvs, step.Env)).To(BeFalse())
					}
				}
			})

			It("should override Build env vars with BuildRun env vars in every step", func() {
				got, err = resources.GenerateTaskSpec(config.NewDefaultConfig(), buildWithEnvs, buildRunWithEnvs, buildStrategy.Spec.Steps, []buildv1beta1.Parameter{}, buildStrategy.GetVolumes())
				Expect(err).To(BeNil())

				combinedEnvs, err := env.MergeEnvVars(buildRunWithEnvs.Spec.Env, buildWithEnvs.Spec.Env, true)
				Expect(err).NotTo(HaveOccurred())

				for _, step := range got.Steps {
					if _, ok := buildStrategyStepNames[step.Name]; ok {
						Expect(len(step.Env)).To(Equal(len(combinedEnvs)))
						Expect(reflect.DeepEqual(combinedEnvs, step.Env)).To(BeTrue())
					} else {
						Expect(reflect.DeepEqual(combinedEnvs, step.Env)).To(BeFalse())
					}

				}
			})

			It("should fail attempting to override an env var in a BuildStrategy", func() {
				got, err = resources.GenerateTaskSpec(config.NewDefaultConfig(), buildWithEnvs, buildRunWithEnvs, buildStrategyWithEnvs.Spec.Steps, []buildv1beta1.Parameter{}, buildStrategy.GetVolumes())
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("error(s) occurred merging environment variables into BuildStrategy \"buildah\" steps: [environment variable \"MY_VAR_1\" already exists, environment variable \"MY_VAR_2\" already exists]"))
			})
		})

		Context("when only BuildRun has output image labels and annotation defined ", func() {
			BeforeEach(func() {
				build, err = ctl.LoadBuildYAML([]byte(test.BuildahBuildWithOutput))
				Expect(err).To(BeNil())

				buildRun, err = ctl.LoadBuildRunFromBytes([]byte(test.BuildahBuildRunWithOutputImageLabelsAndAnnotations))
				Expect(err).To(BeNil())

				buildStrategy, err = ctl.LoadBuildStrategyFromBytes([]byte(test.MinimalBuildahBuildStrategy))
				Expect(err).To(BeNil())
				buildStrategy.Spec.Steps[0].ImagePullPolicy = "Always"

				expectedCommandOrArg = []string{
					"bud", "--tag=$(params.shp-output-image)", fmt.Sprintf("--file=$(inputs.params.%s)", "DOCKERFILE"), "$(params.shp-source-context)",
				}

				JustBeforeEach(func() {
					got, err = resources.GenerateTaskSpec(config.NewDefaultConfig(), build, buildRun, buildStrategy.Spec.Steps, []buildv1beta1.Parameter{}, buildStrategy.GetVolumes())
					Expect(err).To(BeNil())
				})

				It("should contain an image-processing step to mutate the image with labels and annotations merged from build and buildrun", func() {
					Expect(got.Steps[3].Name).To(Equal("image-processing"))
					Expect(got.Steps[3].Command[0]).To(Equal("/ko-app/image-processing"))
					Expect(got.Steps[3].Args).To(Equal([]string{
						"--image",
						"$(params.shp-output-image)",
						"--result-file-image-digest",
						"$(results.shp-image-digest.path)",
						"result-file-image-size",
						"$(results.shp-image-size.path)",
						"--annotation",
						"org.opencontainers.owner=my-company",
						"--label",
						"maintainer=new-team@my-company.com",
						"foo=bar",
					}))
				})
			})
		})

		Context("when Build and BuildRun both have output image labels and annotation defined ", func() {
			BeforeEach(func() {
				build, err = ctl.LoadBuildYAML([]byte(test.BuildahBuildWithAnnotationAndLabel))
				Expect(err).To(BeNil())

				buildRun, err = ctl.LoadBuildRunFromBytes([]byte(test.BuildahBuildRunWithOutputImageLabelsAndAnnotations))
				Expect(err).To(BeNil())

				buildStrategy, err = ctl.LoadBuildStrategyFromBytes([]byte(test.MinimalBuildahBuildStrategy))
				Expect(err).To(BeNil())
				buildStrategy.Spec.Steps[0].ImagePullPolicy = "Always"

				expectedCommandOrArg = []string{
					"bud", "--tag=$(params.shp-output-image)", fmt.Sprintf("--file=$(inputs.params.%s)", "DOCKERFILE"), "$(params.shp-source-context)",
				}

				JustBeforeEach(func() {
					got, err = resources.GenerateTaskSpec(config.NewDefaultConfig(), build, buildRun, buildStrategy.Spec.Steps, []buildv1beta1.Parameter{}, buildStrategy.GetVolumes())
					Expect(err).To(BeNil())
				})

				It("should contain an image-processing step to mutate the image with labels and annotations merged from build and buildrun", func() {
					Expect(got.Steps[3].Name).To(Equal("image-processing"))
					Expect(got.Steps[3].Command[0]).To(Equal("/ko-app/image-processing"))
					Expect(got.Steps[3].Args).To(Equal([]string{
						"--image",
						"$(params.shp-output-image)",
						"--result-file-image-digest",
						"$(results.shp-image-digest.path)",
						"result-file-image-size",
						"$(results.shp-image-size.path)",
						"--annotation",
						"org.opencontainers.owner=my-company",
						"org.opencontainers.image.url=https://my-company.com/images",
						"--label",
						"maintainer=new-team@my-company.com",
						"foo=bar",
					}))
				})
			})
		})

		Context("when Build and BuildRun have no source", func() {

			BeforeEach(func() {
				build, err = ctl.LoadBuildYAML([]byte(test.BuildBSMinimalNoSource))
				Expect(err).ToNot(HaveOccurred())

				buildRun, err = ctl.LoadBuildRunFromBytes([]byte(test.MinimalBuildahBuildRun))
				Expect(err).ToNot(HaveOccurred())

				buildStrategy, err = ctl.LoadBuildStrategyFromBytes([]byte(test.MinimalBuildahBuildStrategy))
				Expect(err).ToNot(HaveOccurred())
			})

			JustBeforeEach(func() {
				taskRun, err := resources.GenerateTaskRun(config.NewDefaultConfig(), build, buildRun, "", buildStrategy)
				Expect(err).ToNot(HaveOccurred())
				got = taskRun.Spec.TaskSpec
			})

			It("should not contain a source step", func() {
				sourceStepFound := false
				for _, step := range got.Steps {
					if strings.HasPrefix(step.Name, "source") {
						sourceStepFound = true
					}
				}
				Expect(sourceStepFound).To(BeFalse(), "Found unexpected source step")
			})
		})
	})

	Describe("Generate the TaskRun", func() {
		var (
			k8sDuration30s                                                *metav1.Duration
			k8sDuration1m                                                 *metav1.Duration
			namespace, outputPath, outputPathBuildRun, serviceAccountName string
			got                                                           *pipelineapi.TaskRun
			err                                                           error
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
				Expect(got.Labels[buildv1beta1.LabelBuild]).To(Equal(build.Name))
				Expect(got.Labels[buildv1beta1.LabelBuildRun]).To(Equal(buildRun.Name))
				Expect(got.Labels[buildv1beta1.LabelBuildStrategyName]).To(Equal(build.Spec.Strategy.Name))
				Expect(got.Labels[buildv1beta1.LabelBuildStrategyGeneration]).To(Equal("0"))
			})

			It("should filter out certain annotations when propagating them to the TaskRun", func() {
				Expect(len(got.Annotations)).To(Equal(2))
				Expect(got.Annotations["kubernetes.io/egress-bandwidth"]).To(Equal("1M"))
				Expect(got.Annotations["kubernetes.io/ingress-bandwidth"]).To(Equal("1M"))
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
				Expect(got.Spec.TaskSpec.Steps[1].ComputeResources).To(Equal(expectedResourceOrArg))
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
				Expect(got.Labels[buildv1beta1.LabelBuild]).To(Equal(build.Name))
				Expect(got.Labels[buildv1beta1.LabelBuildRun]).To(Equal(buildRun.Name))
			})

			It("should ensure generated TaskRun's spec special input params are correct", func() {
				params := got.Spec.Params

				paramSourceRootFound := false
				paramSourceContextFound := false
				paramOutputImageFound := false
				paramOutputInsecureFound := false

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

					case "shp-output-insecure":
						paramOutputInsecureFound = true
						Expect(param.Value.StringVal).To(Equal("false"))

					default:
						Fail(fmt.Sprintf("Unexpected param found: %s", param.Name))
					}
				}

				Expect(paramSourceRootFound).To(BeTrue())
				Expect(paramSourceContextFound).To(BeTrue())
				Expect(paramOutputImageFound).To(BeTrue())
				Expect(paramOutputInsecureFound).To(BeTrue())
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
				Expect(got.Spec.TaskSpec.Steps[1].ComputeResources).To(Equal(expectedResourceOrArg))
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

		Context("when the build and buildrun both specify a nodeSelector", func() {
			BeforeEach(func() {
				build, err = ctl.LoadBuildYAML([]byte(test.MinimalBuildWithNodeSelector))
				Expect(err).To(BeNil())

				buildRun, err = ctl.LoadBuildRunFromBytes([]byte(test.MinimalBuildRunWithNodeSelector))
				Expect(err).To(BeNil())

				buildStrategy, err = ctl.LoadBuildStrategyFromBytes([]byte(test.ClusterBuildStrategyNoOp))
				Expect(err).To(BeNil())
			})

			JustBeforeEach(func() {
				got, err = resources.GenerateTaskRun(config.NewDefaultConfig(), build, buildRun, serviceAccountName, buildStrategy)
				Expect(err).To(BeNil())
			})

			It("should give precedence to the nodeSelector specified in the buildRun", func() {
				Expect(got.Spec.PodTemplate.NodeSelector).To(Equal(buildRun.Spec.NodeSelector))
			})
		})

		Context("when the build and buildrun both specify a Toleration", func() {
			BeforeEach(func() {
				build, err = ctl.LoadBuildYAML([]byte(test.MinimalBuildWithToleration))
				Expect(err).To(BeNil())

				buildRun, err = ctl.LoadBuildRunFromBytes([]byte(test.MinimalBuildRunWithToleration))
				Expect(err).To(BeNil())

				buildStrategy, err = ctl.LoadBuildStrategyFromBytes([]byte(test.ClusterBuildStrategyNoOp))
				Expect(err).To(BeNil())
			})

			JustBeforeEach(func() {
				got, err = resources.GenerateTaskRun(config.NewDefaultConfig(), build, buildRun, serviceAccountName, buildStrategy)
				Expect(err).To(BeNil())
			})

			It("should give precedence to the Toleration values specified in the buildRun", func() {
				Expect(got.Spec.PodTemplate.Tolerations[0].Key).To(Equal(buildRun.Spec.Tolerations[0].Key))
				Expect(got.Spec.PodTemplate.Tolerations[0].Operator).To(Equal(buildRun.Spec.Tolerations[0].Operator))
				Expect(got.Spec.PodTemplate.Tolerations[0].Value).To(Equal(buildRun.Spec.Tolerations[0].Value))
			})
		})
	})
})
