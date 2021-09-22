// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main_test

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/shipwright-io/build/cmd/bundle"

	"github.com/google/go-containerregistry/pkg/name"
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

var _ = Describe("Bundle Loader", func() {
	var run = func(args ...string) error {
		log.SetOutput(ioutil.Discard)
		os.Args = append([]string{"tool"}, args...)
		return Do(context.Background())
	}

	var withTempDir = func(f func(target string)) {
		path, err := ioutil.TempDir(os.TempDir(), "bundle")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(path)

		f(path)
	}

	var withTempFile = func(pattern string, f func(filename string)) {
		file, err := ioutil.TempFile(os.TempDir(), pattern)
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove(file.Name())

		f(file.Name())
	}

	var filecontent = func(path string) string {
		data, err := ioutil.ReadFile(path)
		Expect(err).ToNot(HaveOccurred())
		return string(data)
	}

	var getImage = func(tag name.Tag) containerreg.Image {
		ref, err := name.ParseReference(tag.String())
		Expect(err).To(BeNil())

		desc, err := remote.Get(ref)
		Expect(err).To(BeNil())

		img, err := desc.Image()
		Expect(err).To(BeNil())

		return img
	}

	var getImageDigest = func(tag name.Tag) containerreg.Hash {
		digest, err := getImage(tag).Digest()
		Expect(err).To(BeNil())

		return digest
	}

	Context("Error cases", func() {
		It("should fail in case the image is not specified", func() {
			Expect(run(
				"--image", "",
			)).To(HaveOccurred())
		})
	})

	Context("Pulling image anonymously", func() {
		const exampleImage = "quay.io/shipwright/source-bundle:latest"

		It("should pull and unbundle an image from a public registry", func() {
			withTempDir(func(target string) {
				Expect(run(
					"--image", exampleImage,
					"--target", target,
				)).ToNot(HaveOccurred())

				Expect(filepath.Join(target, "LICENSE")).To(BeAnExistingFile())
			})
		})

		It("should store image digest into file specified in --result-file-image-digest flags", func() {
			withTempDir(func(target string) {
				withTempFile("image-digest", func(filename string) {
					Expect(run(
						"--image", exampleImage,
						"--target", target,
						"--result-file-image-digest",
						filename,
					)).ToNot(HaveOccurred())

					tag, err := name.NewTag(exampleImage)
					Expect(err).To(BeNil())

					Expect(filecontent(filename)).To(Equal(getImageDigest(tag).String()))
				})
			})
		})
	})
})
