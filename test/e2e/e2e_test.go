// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

var _ = Describe("For a Kubernetes cluster with Tekton and build installed", func() {
	var (
		testID string
		err    error

		build    *buildv1alpha1.Build
		buildRun *buildv1alpha1.BuildRun
	)

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
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

	Context("when a Buildah build is defined", func() {

		BeforeEach(func() {
			testID = generateTestID("buildah")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_buildah_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_buildah_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			validateBuildRunToSucceed(testBuild, buildRun)
		})
	})

	Context("when a Buildah build with a contextDir and a custom Dockerfile name is defined", func() {

		BeforeEach(func() {
			testID = generateTestID("buildah-custom-context-dockerfile")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/build_buildah_cr_custom_context+dockerfile.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_buildah_cr_custom_context+dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			validateBuildRunToSucceed(testBuild, buildRun)
		})
	})

	Context("when a heroku Buildpacks build is defined using a cluster strategy", func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-heroku")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_buildpacks-v3-heroku_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_buildpacks-v3-heroku_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			validateBuildRunToSucceed(testBuild, buildRun)
		})
	})

	Context("when a heroku Buildpacks build is defined using a namespaced strategy", func() {
		var buildStrategy *buildv1alpha1.BuildStrategy

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-heroku-namespaced")

			buildStrategy, err = buildStrategyTestData(testBuild.Namespace, "samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_namespaced_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			err = testBuild.CreateBuildStrategy(buildStrategy)
			Expect(err).ToNot(HaveOccurred())

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_buildpacks-v3-heroku_namespaced_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_buildpacks-v3-heroku_namespaced_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(testBuild, buildRun)
		})

		AfterEach(func() {
			err = testBuild.DeleteBuildStrategy(buildStrategy.Name)
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
				"samples/build/build_buildpacks-v3_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_buildpacks-v3_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			validateBuildRunToSucceed(testBuild, buildRun)
		})
	})

	Context("when a Buildpacks v3 build is defined using a namespaced strategy", func() {
		var buildStrategy *buildv1alpha1.BuildStrategy

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-namespaced")

			buildStrategy, err = buildStrategyTestData(testBuild.Namespace, "samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_namespaced_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			err = testBuild.CreateBuildStrategy(buildStrategy)
			Expect(err).ToNot(HaveOccurred())

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_buildpacks-v3_namespaced_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_buildpacks-v3_namespaced_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			validateBuildRunToSucceed(testBuild, buildRun)
		})

		AfterEach(func() {
			err = testBuild.DeleteBuildStrategy(buildStrategy.Name)
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
				"test/data/build_buildpacks-v3_php_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_buildpacks-v3_php_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			validateBuildRunToSucceed(testBuild, buildRun)
		})
	})

	Context("when a Buildpacks v3 build is defined for a ruby runtime", func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-ruby")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/build_buildpacks-v3_ruby_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_buildpacks-v3_ruby_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(testBuild, buildRun)
		})
	})

	Context("when a Buildpacks v3 build is defined for a golang runtime", func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-golang")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/build_buildpacks-v3_golang_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_buildpacks-v3_golang_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			validateBuildRunToSucceed(testBuild, buildRun)
		})
	})

	Context("when a build uses the build-run-deletion annotation", func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-golang")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/build_buildpacks-v3_golang_delete_cr.yaml",
			)
		})

		It("successfully deletes the BuildRun after the Build is deleted", func() {
			By("running a build and expecting it to succeed")
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_buildpacks-v3_golang_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			validateBuildRunToSucceed(testBuild, buildRun)

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
				"test/data/build_buildpacks-v3_java_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_buildpacks-v3_java_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(testBuild, buildRun)
		})
	})

	Context("when a Kaniko build is defined to use public GitHub", func() {

		BeforeEach(func() {
			testID = generateTestID("kaniko")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_kaniko_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_kaniko_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(testBuild, buildRun)
		})
	})

	Context("when a Kaniko build with a Dockerfile that requires advanced permissions is defined", func() {

		BeforeEach(func() {
			testID = generateTestID("kaniko-advanced-dockerfile")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/build_kaniko_cr_advanced_dockerfile.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_kaniko_cr_advanced_dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			validateBuildRunToSucceed(testBuild, buildRun)
		})
	})

	Context("when a Kaniko build with a contextDir and a custom Dockerfile name is defined", func() {

		BeforeEach(func() {
			testID = generateTestID("kaniko-custom-context-dockerfile")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/build_kaniko_cr_custom_context+dockerfile.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_kaniko_cr_custom_context+dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(testBuild, buildRun)
		})
	})

	Context("when a Kaniko+Trivy build is defined to use an image with no critical CVEs", func() {

		BeforeEach(func() {
			testID = generateTestID("kaniko-trivy-good")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_kaniko-trivy-good_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_kaniko-trivy-good_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(testBuild, buildRun)
		})
	})

	Context("when a Kaniko+Trivy build is defined to use an image with a critical CVE", func() {

		BeforeEach(func() {
			testID = generateTestID("kaniko-trivy-bad")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_kaniko-trivy-bad_cr.yaml",
			)
		})

		It("fails to run a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_kaniko-trivy-bad_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToFail(testBuild, buildRun)
		})
	})

	Context("when a Buildkit build with a contextDir and a path to a Dockerfile is defined", func() {

		BeforeEach(func() {
			testID = generateTestID("buildkit-custom-context")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/build_buildkit_cr_insecure_registry.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_buildkit_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(testBuild, buildRun)
		})
	})

	Context("when a s2i build is defined", func() {

		BeforeEach(func() {
			testID = generateTestID("s2i")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_source-to-image_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_source-to-image_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			validateBuildRunToSucceed(testBuild, buildRun)
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
					"test/data/build_buildah_cr_private_github.yaml",
				)
			})

			It("successfully runs a build", func() {
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_buildah_cr.yaml")
				Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

				validateBuildRunToSucceed(testBuild, buildRun)
			})
		})

		Context("when a Buildah build is defined to use a private GitLab repository", func() {

			BeforeEach(func() {
				testID = generateTestID("private-gitlab-buildah")

				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/build_buildah_cr_private_gitlab.yaml",
				)
			})

			It("successfully runs a build", func() {
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_buildah_cr.yaml")
				Expect(err).ToNot(HaveOccurred())

				validateBuildRunToSucceed(testBuild, buildRun)
			})
		})

		Context("when a Kaniko build is defined to use a private GitHub repository", func() {

			BeforeEach(func() {
				testID = generateTestID("private-github-kaniko")

				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/build_kaniko_cr_private_github.yaml",
				)
			})

			It("successfully runs a build", func() {
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_kaniko_cr.yaml")
				Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

				validateBuildRunToSucceed(testBuild, buildRun)
			})
		})

		Context("when a Kaniko build is defined to use a private GitLab repository", func() {

			BeforeEach(func() {
				testID = generateTestID("private-gitlab-kaniko")

				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/build_kaniko_cr_private_gitlab.yaml",
				)
			})

			It("successfully runs a build", func() {
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_kaniko_cr.yaml")
				Expect(err).ToNot(HaveOccurred())

				validateBuildRunToSucceed(testBuild, buildRun)
			})
		})

		Context("when a s2i build is defined to use a private GitHub repository", func() {

			BeforeEach(func() {
				testID = generateTestID("private-github-s2i")

				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/build_source-to-image_cr_private_github.yaml",
				)
			})

			It("successfully runs a build", func() {
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_source-to-image_cr.yaml")
				Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

				validateBuildRunToSucceed(testBuild, buildRun)
			})
		})
	})

	Context("when results from TaskRun surface to BuildRun", func() {
		BeforeEach(func() {
			testID = generateTestID("kaniko")
		})

		It("should BuildRun status field contains results from default(git) step", func() {
			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_kaniko_cr.yaml",
			)

			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_kaniko_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
		})

		It("should BuildRun status field contains results from default(bundle) step", func() {
			inputImage := "quay.io/shipwright/source-bundle:latest"
			outputImage := fmt.Sprintf("%s/%s:%s",
				os.Getenv(EnvVarImageRepo),
				testID,
				"latest",
			)

			build, err = NewBuildPrototype().
				ClusterBuildStrategy("kaniko").
				Name(testID).
				Namespace(testBuild.Namespace).
				SourceBundle(inputImage).
				SourceContextDir("docker-build").
				Dockerfile("Dockerfile").
				OutputImage(outputImage).
				OutputImageCredentials(os.Getenv(EnvVarImageRepoSecret)).
				Create()
			Expect(err).ToNot(HaveOccurred())

			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				ForBuild(build).
				GenerateServiceAccount().
				Create()
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromBundleSource(buildRun)
		})
	})
})
