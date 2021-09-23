// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/sources"

	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Git", func() {

	cfg := config.NewDefaultConfig()

	Context("when adding a public Git source", func() {

		var taskSpec *tektonv1beta1.TaskSpec

		BeforeEach(func() {
			taskSpec = &tektonv1beta1.TaskSpec{}
		})

		JustBeforeEach(func() {
			sources.AppendGitStep(cfg, taskSpec, buildv1alpha1.Source{
				URL: "https://github.com/shipwright-io/build",
			}, "default")
		})

		It("adds results for the commit sha and commit author", func() {
			Expect(len(taskSpec.Results)).To(Equal(2))
			Expect(taskSpec.Results[0].Name).To(Equal("shp-source-default-commit-sha"))
			Expect(taskSpec.Results[1].Name).To(Equal("shp-source-default-commit-author"))
		})

		It("adds a step", func() {
			Expect(len(taskSpec.Steps)).To(Equal(1))
			Expect(taskSpec.Steps[0].Name).To(Equal("source-default"))
			Expect(taskSpec.Steps[0].Image).To(Equal(cfg.GitContainerTemplate.Image))
			Expect(taskSpec.Steps[0].Args).To(Equal([]string{
				"--url",
				"https://github.com/shipwright-io/build",
				"--target",
				"$(params.shp-source-root)",
				"--result-file-commit-sha",
				"$(results.shp-source-default-commit-sha.path)",
				"--result-file-commit-author",
				"$(results.shp-source-default-commit-author.path)",
			}))
		})
	})

	Context("when adding a private Git source", func() {

		var taskSpec *tektonv1beta1.TaskSpec

		BeforeEach(func() {
			taskSpec = &tektonv1beta1.TaskSpec{}
		})

		JustBeforeEach(func() {
			sources.AppendGitStep(cfg, taskSpec, buildv1alpha1.Source{
				URL: "git@github.com:shipwright-io/build.git",
				Credentials: &corev1.LocalObjectReference{
					Name: "a.secret",
				},
			}, "default")
		})

		It("adds results for the commit sha and commit author", func() {
			Expect(len(taskSpec.Results)).To(Equal(2))
			Expect(taskSpec.Results[0].Name).To(Equal("shp-source-default-commit-sha"))
			Expect(taskSpec.Results[1].Name).To(Equal("shp-source-default-commit-author"))
		})

		It("adds a volume for the secret", func() {
			Expect(len(taskSpec.Volumes)).To(Equal(1))
			Expect(taskSpec.Volumes[0].Name).To(Equal("shp-a-secret"))
			Expect(taskSpec.Volumes[0].VolumeSource.Secret).NotTo(BeNil())
			Expect(taskSpec.Volumes[0].VolumeSource.Secret.SecretName).To(Equal("a.secret"))
		})

		It("adds a step", func() {
			Expect(len(taskSpec.Steps)).To(Equal(1))
			Expect(taskSpec.Steps[0].Name).To(Equal("source-default"))
			Expect(taskSpec.Steps[0].Image).To(Equal(cfg.GitContainerTemplate.Image))
			Expect(taskSpec.Steps[0].Args).To(Equal([]string{
				"--url",
				"git@github.com:shipwright-io/build.git",
				"--target",
				"$(params.shp-source-root)",
				"--result-file-commit-sha",
				"$(results.shp-source-default-commit-sha.path)",
				"--result-file-commit-author",
				"$(results.shp-source-default-commit-author.path)",
				"--secret-path",
				"/workspace/shp-source-secret",
			}))
			Expect(len(taskSpec.Steps[0].VolumeMounts)).To(Equal(1))
			Expect(taskSpec.Steps[0].VolumeMounts[0].Name).To(Equal("shp-a-secret"))
			Expect(taskSpec.Steps[0].VolumeMounts[0].MountPath).To(Equal("/workspace/shp-source-secret"))
			Expect(taskSpec.Steps[0].VolumeMounts[0].ReadOnly).To(BeTrue())
		})
	})
})
