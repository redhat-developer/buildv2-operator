// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"fmt"
	"log"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/shipwright-io/build/pkg/bundle"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"k8s.io/apimachinery/pkg/util/rand"
)

var _ = Describe("Bundle", func() {
	withTempDir := func(f func(tempDir string)) {
		tempDir, err := os.MkdirTemp("", "bundle")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(tempDir)
		f(tempDir)
	}

	withTempRegistry := func(f func(endpoint string)) {
		logLogger := log.Logger{}
		logLogger.SetOutput(GinkgoWriter)

		s := httptest.NewServer(
			registry.New(
				registry.Logger(&logLogger),
				registry.WithReferrersSupport(true),
			),
		)
		defer s.Close()

		u, err := url.Parse(s.URL)
		Expect(err).ToNot(HaveOccurred())

		f(u.Host)
	}

	Context("packing and unpacking", func() {
		It("should pack and unpack a directory", func() {
			withTempDir(func(tempDir string) {
				r, err := Pack(filepath.Join("..", "..", "test", "bundle"))
				Expect(err).ToNot(HaveOccurred())
				Expect(r).ToNot(BeNil())

				details, err := Unpack(r, tempDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(details).ToNot(BeNil())

				Expect(filepath.Join(tempDir, "README.md")).To(BeAnExistingFile())
				Expect(filepath.Join(tempDir, ".someToolDir", "config.json")).ToNot(BeAnExistingFile())
				Expect(filepath.Join(tempDir, "somefile")).To(BeAnExistingFile())
				Expect(filepath.Join(tempDir, "linktofile")).To(BeAnExistingFile())
			})
		})

		It("should pack and unpack a directory including all files even if a sub-directory is non-writable", func() {
			withTempDir(func(source string) {
				// Setup a case where there are files in a directory, which is non-writable
				Expect(os.Mkdir(filepath.Join(source, "some-dir"), os.FileMode(0777))).To(Succeed())
				Expect(os.WriteFile(filepath.Join(source, "some-dir", "some-file"), []byte(`foobar`), os.FileMode(0644))).To(Succeed())
				Expect(os.Chmod(filepath.Join(source, "some-dir"), os.FileMode(0555))).To(Succeed())

				r, err := Pack(source)
				Expect(err).ToNot(HaveOccurred())
				Expect(r).ToNot(BeNil())

				withTempDir(func(target string) {
					details, err := Unpack(r, target)
					Expect(err).ToNot(HaveOccurred())
					Expect(details).ToNot(BeNil())

					Expect(filepath.Join(target, "some-dir", "some-file")).To(BeAnExistingFile())
				})
			})
		})
	})

	Context("packing/pushing and pulling/unpacking", func() {
		It("should pull and unpack an image", func() {
			withTempRegistry(func(endpoint string) {
				ref, err := name.ParseReference(fmt.Sprintf("%s/namespace/unit-test-pkg-bundle-%s:latest", endpoint, rand.String(5)))
				Expect(err).ToNot(HaveOccurred())

				By("packing and pushing an image", func() {
					_, err := PackAndPush(ref, filepath.Join("..", "..", "test", "bundle"))
					Expect(err).ToNot(HaveOccurred())
				})

				By("pulling and unpacking the image", func() {
					withTempDir(func(tempDir string) {
						image, err := PullAndUnpack(ref, tempDir)
						Expect(err).ToNot(HaveOccurred())
						Expect(image).ToNot(BeNil())

						Expect(filepath.Join(tempDir, "README.md")).To(BeAnExistingFile())
						Expect(filepath.Join(tempDir, ".someToolDir", "config.json")).ToNot(BeAnExistingFile())
						Expect(filepath.Join(tempDir, "somefile")).To(BeAnExistingFile())
						Expect(filepath.Join(tempDir, "linktofile")).To(BeAnExistingFile())
					})
				})
			})
		})
	})
})
