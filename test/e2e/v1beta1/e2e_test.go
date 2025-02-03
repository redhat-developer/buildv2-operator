// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"os"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	shpgit "github.com/shipwright-io/build/pkg/git"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("For a Kubernetes cluster with Tekton and build installed", func() {
	var (
		testID string
		err    error

		build    *buildv1beta1.Build
		buildRun *buildv1beta1.BuildRun
	)

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			printTestFailureDebugInfo(testBuild, testBuild.Namespace, testID)

		} else if buildRun != nil {
			validateServiceAccountDeletion(buildRun, testBuild.Namespace)
		}

		if buildRun != nil {
			testBuild.DeleteBR(buildRun.Name)
			buildRun = nil
		}

		if build != nil {
			testBuild.DeleteBuild(build.Name)
			build = nil
		}
	})

	Context("when a Buildah build is defined that is using shipwright-managed push", func() {

		BeforeEach(func() {
			testID = generateTestID("buildah")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1beta1/build/build_buildah_shipwright_managed_push_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1beta1/buildrun/buildrun_buildah_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")
			appendRegistryInsecureParamValue(build, buildRun)

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildah build is defined that is using strategy-managed push", func() {

		BeforeEach(func() {
			testID = generateTestID("buildah")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1beta1/build/build_buildah_strategy_managed_push_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1beta1/buildrun/buildrun_buildah_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")
			appendRegistryInsecureParamValue(build, buildRun)

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildah build with a contextDir and a custom Dockerfile name is defined", func() {

		BeforeEach(func() {
			testID = generateTestID("buildah-custom-context-dockerfile")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_buildah_cr_custom_context+dockerfile.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_buildah_cr_custom_context+dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")
			appendRegistryInsecureParamValue(build, buildRun)

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a BuildAh build runs with a custom target stage", func() {

		BeforeEach(func() {
			testID = generateTestID("buildah-custom-target-stage")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_buildah_cr_target_stage.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_buildah_cr_target_stage.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a heroku Buildpacks build is defined using a cluster strategy", func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-heroku")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1beta1/build/build_buildpacks-v3-heroku_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1beta1/buildrun/buildrun_buildpacks-v3-heroku_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a heroku Buildpacks build is defined using a namespaced strategy", func() {
		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-heroku-namespaced")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1beta1/build/build_buildpacks-v3-heroku_namespaced_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1beta1/buildrun/buildrun_buildpacks-v3-heroku_namespaced_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})

		AfterEach(func() {
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when a Buildpacks v3 build is defined using a cluster strategy", func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1beta1/build/build_buildpacks-v3_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1beta1/buildrun/buildrun_buildpacks-v3_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildpacks v3 build is defined using a namespaced strategy", func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-namespaced")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1beta1/build/build_buildpacks-v3_namespaced_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1beta1/buildrun/buildrun_buildpacks-v3_namespaced_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})

		AfterEach(func() {
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when a Buildpacks v3 build is defined for a php runtime", func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-php")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_buildpacks-v3_php_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_buildpacks-v3_php_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildpacks v3 build is defined for a ruby runtime", func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-ruby")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_buildpacks-v3_ruby_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_buildpacks-v3_ruby_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildpacks v3 build is defined for a golang runtime", func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-golang")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_buildpacks-v3_golang_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_buildpacks-v3_golang_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildpacks v3 build is defined for a golang runtime with `BP_GO_TARGETS` env", func() {
		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-golang")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_buildpacks-v3_golang_cr_env.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_buildpacks-v3_golang_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a build uses the build-run-deletion annotation", func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-golang")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_buildpacks-v3_golang_delete_cr.yaml",
			)
		})

		It("successfully deletes the BuildRun after the Build is deleted", func() {
			By("running a build and expecting it to succeed")
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_buildpacks-v3_golang_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)

			By("deleting the parent Build object")
			err = testBuild.DeleteBuild(build.Name)
			Expect(err).NotTo(HaveOccurred(), "error deleting the parent Build")
			Eventually(func() bool {
				_, err = testBuild.GetBR(buildRun.Name)
				if err == nil {
					return false
				}
				if !errors.IsNotFound(err) {
					return false
				}
				return true
			}).Should(BeTrue())
		})
	})

	Context("when a Buildpacks v3 build is defined for a java runtime", func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-java")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_buildpacks-v3_java_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_buildpacks-v3_java_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Kaniko build is defined to use public GitHub", func() {

		BeforeEach(func() {
			testID = generateTestID("kaniko")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1beta1/build/build_kaniko_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1beta1/buildrun/buildrun_kaniko_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Kaniko build with a Dockerfile that requires advanced permissions is defined", func() {

		BeforeEach(func() {
			testID = generateTestID("kaniko-advanced-dockerfile")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_kaniko_cr_advanced_dockerfile.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_kaniko_cr_advanced_dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Kaniko build with a contextDir and a custom Dockerfile name is defined", func() {

		BeforeEach(func() {
			testID = generateTestID("kaniko-custom-context-dockerfile")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_kaniko_cr_custom_context+dockerfile.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_kaniko_cr_custom_context+dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Kaniko build runs with a custom target stage", func() {

		BeforeEach(func() {
			testID = generateTestID("kaniko-custom-target-stage")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_kaniko_cr_target_stage.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_kaniko_cr_target_stage.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildkit build with a contextDir and a path to a Dockerfile is defined", func() {

		BeforeEach(func() {
			testID = generateTestID("buildkit-custom-context")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1beta1/build/build_buildkit_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1beta1/buildrun/buildrun_buildkit_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImagePlatformsExist(buildRun, []v1.Platform{
				{
					Architecture: "amd64",
					OS:           "linux",
				},
				{
					Architecture: "arm64",
					OS:           "linux",
				},
			})
		})
	})

	Context("when a BuildKit build runs with a custom target stage", func() {

		BeforeEach(func() {
			testID = generateTestID("buildkit-custom-target-stage")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_buildkit_cr_target_stage.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_buildkit_cr_target_stage.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Multiarch Native Buildah build is defined", func() {

		BeforeEach(func() {
			testID = generateTestID("buildah-multi-arch-native")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_multiarch_native_buildah_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_multiarch_native_buildah_cr.yaml")
			Expect(err).ToNot(HaveOccurred())
			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
		})
	})
	Context("when a s2i build is defined", func() {

		BeforeEach(func() {
			testID = generateTestID("s2i")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1beta1/build/build_source-to-image_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1beta1/buildrun/buildrun_source-to-image_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a private source repository is used", func() {

		BeforeEach(func() {
			if os.Getenv(EnvVarEnablePrivateRepos) != "true" {
				Skip("Skipping test cases that use a private source repository")
			}
		})

		Context("when a Buildah build is defined to use a private GitHub repository", func() {

			BeforeEach(func() {
				testID = generateTestID("private-github-buildah")

				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/v1beta1/build_buildah_cr_private_github.yaml",
				)
			})

			It("successfully runs a build", func() {
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1beta1/buildrun/buildrun_buildah_cr.yaml")
				Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

				buildRun = validateBuildRunToSucceed(testBuild, buildRun)
				testBuild.ValidateImageDigest(buildRun)
			})
		})

		Context("when a Buildah build is defined to use a private GitLab repository", func() {

			BeforeEach(func() {
				testID = generateTestID("private-gitlab-buildah")

				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/v1beta1/build_buildah_cr_private_gitlab.yaml",
				)
			})

			It("successfully runs a build", func() {
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1beta1/buildrun/buildrun_buildah_cr.yaml")
				Expect(err).ToNot(HaveOccurred())

				buildRun = validateBuildRunToSucceed(testBuild, buildRun)
				testBuild.ValidateImageDigest(buildRun)
			})
		})

		Context("when a Kaniko build is defined to use a private GitHub repository", func() {

			BeforeEach(func() {
				testID = generateTestID("private-github-kaniko")

				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/v1beta1/build_kaniko_cr_private_github.yaml",
				)
			})

			It("successfully runs a build", func() {
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1beta1/buildrun/buildrun_kaniko_cr.yaml")
				Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

				buildRun = validateBuildRunToSucceed(testBuild, buildRun)
				testBuild.ValidateImageDigest(buildRun)
			})
		})

		Context("when a Kaniko build is defined to use a private GitLab repository", func() {

			BeforeEach(func() {
				testID = generateTestID("private-gitlab-kaniko")

				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/v1beta1/build_kaniko_cr_private_gitlab.yaml",
				)
			})

			It("successfully runs a build", func() {
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1beta1/buildrun/buildrun_kaniko_cr.yaml")
				Expect(err).ToNot(HaveOccurred())

				buildRun = validateBuildRunToSucceed(testBuild, buildRun)
				testBuild.ValidateImageDigest(buildRun)
			})
		})

		Context("when a s2i build is defined to use a private GitHub repository", func() {

			BeforeEach(func() {
				testID = generateTestID("private-github-s2i")

				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/v1beta1/build_source-to-image_cr_private_github.yaml",
				)
			})

			It("successfully runs a build", func() {
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1beta1/buildrun/buildrun_source-to-image_cr.yaml")
				Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

				buildRun = validateBuildRunToSucceed(testBuild, buildRun)
				testBuild.ValidateImageDigest(buildRun)
			})
		})
	})

	Context("when a s2i build uses a non-existent git repository as source", func() {
		It("fails because of prompted authentication which surfaces the to the BuildRun", func() {
			testID = generateTestID("s2i-failing")

			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_non_existing_repo.yaml",
			)

			buildRun, err = buildRunTestData(build.Namespace, testID, "test/data/v1beta1/buildrun_non_existing_repo.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToFail(testBuild, buildRun)
			buildRun, err = testBuild.LookupBuildRun(types.NamespacedName{Name: buildRun.Name, Namespace: testBuild.Namespace})

			Expect(buildRun.Status.FailureDetails.Message).To(Equal(shpgit.AuthPrompted.ToMessage()))
			Expect(buildRun.Status.FailureDetails.Reason).To(Equal(shpgit.AuthPrompted.String()))
		})
	})

	Context("when tolerations are specified", Serial, func() {

		BeforeEach(func() {
			testID = generateTestID("tolerations")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_buildah_tolerations_cr.yaml",
			)

			// Add a taint to both of the kind worker nodes
			nodes, err := testBuild.GetNodes()
			Expect(err).ToNot(HaveOccurred())
			for _, node := range nodes.Items {
				if node.Name == "kind-worker" || node.Name == "kind-worker2" {
					taint := corev1.Taint{Key: "test-key", Value: "test-value", Effect: "NoSchedule"}
					err := testBuild.AddNodeTaint(node.Name, &taint)
					Expect(err).ToNot(HaveOccurred())
				}
			}
		})

		AfterEach(func() {
			nodes, err := testBuild.GetNodes()
			Expect(err).ToNot(HaveOccurred())
			for _, node := range nodes.Items {
				if node.Name == "kind-worker" || node.Name == "kind-worker2" {
					err := testBuild.RemoveNodeTaints(node.Name)
					Expect(err).ToNot(HaveOccurred())
				}
			}
		})

		It("successfully runs a build when it tolerates the node taint", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_buildah_tolerations_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
		})

		It("fails to schedule when the build does not tolerate the node taint", func() {
			// set untolerated value and a low timeout since we do not expect this to be scheduled
			build.Spec.Tolerations[0].Value = "untolerated"
			build.Spec.Timeout = &metav1.Duration{Duration: time.Minute}
			testBuild.UpdateBuild(build)

			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_buildah_tolerations_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			validateBuildRunToFail(testBuild, buildRun)
			buildRun, err = testBuild.LookupBuildRun(types.NamespacedName{Name: buildRun.Name, Namespace: testBuild.Namespace})

			// Pod should fail to schedule and the BuildRun should timeout.
			Expect(buildRun.Status.GetCondition(buildv1beta1.Succeeded).Status).To(Equal(corev1.ConditionFalse))
			Expect(buildRun.Status.GetCondition(buildv1beta1.Succeeded).Reason).To(Equal("BuildRunTimeout"))
			Expect(buildRun.Status.GetCondition(buildv1beta1.Succeeded).Message).To(ContainSubstring("failed to finish within"))
		})
	})
})
