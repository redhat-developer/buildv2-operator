// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	"github.com/shipwright-io/build/test"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

var _ = Describe("TaskRun results to BuildRun", func() {
	var ctl test.Catalog

	Context("when a BuildRun complete successfully", func() {
		var (
			br *build.BuildRun
			tr *pipelinev1beta1.TaskRun
		)

		BeforeEach(func() {
			tr = ctl.DefaultTaskRun("foo", "bar")
			br = ctl.DefaultBuildRun("foo", "bar")
		})

		It("should surface the TaskRun results emitting from default(git) source step", func() {
			commitSha := "0e0583421a5e4bf562ffe33f3651e16ba0c78591"

			tr.Status.TaskRunResults = append(tr.Status.TaskRunResults,
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-source-default-commit-sha",
					Value: commitSha,
				},
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-source-default-commit-author",
					Value: "foo bar",
				})

			resources.UpdateBuildRunUsingTaskResults(br, tr)

			Expect(len(br.Status.Sources)).To(Equal(1))
			Expect(br.Status.Sources[0].Git.CommitSha).To(Equal(commitSha))
			Expect(br.Status.Sources[0].Git.CommitAuthor).To(Equal("foo bar"))
		})

		It("should surface the TaskRun results emitting from default(bundle) source step", func() {
			bundleImageDigest := "sha256:fe1b73cd25ac3f11dec752755e2"

			tr.Status.TaskRunResults = append(tr.Status.TaskRunResults,
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-source-default-bundle-image-digest",
					Value: bundleImageDigest,
				})

			resources.UpdateBuildRunUsingTaskResults(br, tr)

			Expect(len(br.Status.Sources)).To(Equal(1))
			Expect(br.Status.Sources[0].Bundle.Digest).To(Equal(bundleImageDigest))
		})

		It("should surface the TaskRun results emitting from output step", func() {
			imageDigest := "sha256:fe1b73cd25ac3f11dec752755e2"

			tr.Status.TaskRunResults = append(tr.Status.TaskRunResults,
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-image-digest",
					Value: imageDigest,
				},
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-image-size",
					Value: "230",
				})

			resources.UpdateBuildRunUsingTaskResults(br, tr)

			Expect(br.Status.Output.Digest).To(Equal(imageDigest))
			Expect(br.Status.Output.Size).To(Equal("230"))
		})

		It("should surface the TaskRun results emitting from source and output step", func() {
			commitSha := "0e0583421a5e4bf562ffe33f3651e16ba0c78591"
			imageDigest := "sha256:fe1b73cd25ac3f11dec752755e2"

			tr.Status.TaskRunResults = append(tr.Status.TaskRunResults,
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-source-default-commit-sha",
					Value: commitSha,
				},
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-source-default-commit-author",
					Value: "foo bar",
				},
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-image-digest",
					Value: imageDigest,
				},
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-image-size",
					Value: "230",
				})

			resources.UpdateBuildRunUsingTaskResults(br, tr)

			Expect(len(br.Status.Sources)).To(Equal(1))
			Expect(br.Status.Sources[0].Git.CommitSha).To(Equal(commitSha))
			Expect(br.Status.Sources[0].Git.CommitAuthor).To(Equal("foo bar"))
			Expect(br.Status.Output.Digest).To(Equal(imageDigest))
			Expect(br.Status.Output.Size).To(Equal("230"))
		})
	})
})
