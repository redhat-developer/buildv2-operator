// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

var _ = Describe("For a Kubernetes cluster with Tekton and build installed", func() {
	var (
		testID   string
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

	Context("when a Buildah build with label and annotation is defined", func() {
		BeforeEach(func() {
			testID = generateTestID("buildah-mutate")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/build_buildah_cr_mutate.yaml",
			)
		})

		It("should mutate an image with annotation and label", func() {
			buildRun, err := buildRunTestData(
				testBuild.Namespace, testID,
				"test/data/buildrun_buildah_cr_mutate.yaml",
			)
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			validateBuildRunToSucceed(testBuild, buildRun)

			Expect(
				getImageAnnotation(getImage(build), "org.opencontainers.image.url"),
			).To(Equal("https://my-company.com/images"))

			Expect(
				getImageLabel(getImage(build), "maintainer"),
			).To(Equal("team@my-company.com"))
		})
	})
})

func getRegistryAuthentication(
	build *buildv1alpha1.Build,
	ref name.Reference,
) authn.Authenticator {
	// In case no secret is mounted, use anonymous
	if build.Spec.Output.Credentials == nil || build.Spec.Output.Credentials.Name == "" {
		log.Printf("No access credentials provided, using anonymous mode")
		return authn.Anonymous
	}

	secret, err := testBuild.LookupSecret(
		types.NamespacedName{
			Namespace: build.Namespace,
			Name:      build.Spec.Output.Credentials.Name,
		},
	)
	Expect(err).ToNot(HaveOccurred(), "Error retrieving registry secret")

	type auth struct {
		Auths map[string]authn.AuthConfig `json:"auths,omitempty"`
	}

	var authConfig auth

	Expect(json.Unmarshal(secret.Data[".dockerconfigjson"], &authConfig)).
		ToNot(HaveOccurred())

	// Look-up the respective registry server inside the credentials
	registryName := ref.Context().RegistryStr()
	if registryName == name.DefaultRegistry {
		registryName = authn.DefaultAuthKey
	}

	return authn.FromConfig(authConfig.Auths[registryName])
}

func getImage(build *buildv1alpha1.Build) containerreg.Image {
	// In the GitHub action, we are using a registry inside the cluster to
	// push the image created by `buildRun`. The registry inside the cluster
	// is not directly accessible from the local, so that we have mapped
	// the cluster registry port to the local system
	// by providing `test/kind/config.yaml` config to the kind
	image := strings.Replace(
		build.Spec.Output.Image,
		"registry.registry.svc.cluster.local",
		"localhost", 1,
	)

	ref, err := name.ParseReference(image)
	Expect(err).To(BeNil())

	img, err := remote.Image(ref, remote.WithAuth(getRegistryAuthentication(build, ref)))
	Expect(err).To(BeNil())

	return img
}

func getImageAnnotation(img containerreg.Image, annotation string) string {
	manifest, err := img.Manifest()
	Expect(err).To(BeNil())

	return manifest.Annotations[annotation]
}

func getImageLabel(img containerreg.Image, label string) string {
	config, err := img.ConfigFile()
	Expect(err).To(BeNil())

	return config.Config.Labels[label]
}
