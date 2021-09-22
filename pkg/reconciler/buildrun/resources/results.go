// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"

	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type sourceResult struct {
	defaultSource,
	bundleSource build.SourceResult
}

const (
	defaultSourceName       = "default"
	commitSHAResult         = "commit-sha"
	commitAuthorResult      = "commit-author"
	bundleImageDigestResult = "bundle-image-digest"
	imageDigestResult       = "image-digest"
	imageSizeResult         = "image-size"
)

// UpdateBuildRunUsingTaskResults surface the task results
// to the buildrun
func UpdateBuildRunUsingTaskResults(
	buildRun *build.BuildRun,
	lastTaskRun *pipeline.TaskRun,
) {
	var sources sourceResult

	// Initializing output result
	buildRun.Status.Output = &build.Output{}

	for _, result := range lastTaskRun.Status.TaskRunResults {
		updateBuildRunStatus(buildRun, result, &sources)
	}

	// Appending the source result only if the defined source
	// from build spec emitting the results
	if sources.defaultSource.Name != "" {
		buildRun.Status.Sources = append(buildRun.Status.Sources, sources.defaultSource)
	}

	if sources.bundleSource.Name != "" {
		buildRun.Status.Sources = append(buildRun.Status.Sources, sources.bundleSource)
	}
}

func updateBuildRunStatus(
	buildRun *build.BuildRun,
	result pipeline.TaskRunResult,
	sources *sourceResult,
) {
	switch result.Name {
	case generateSourceResultName(defaultSourceName, commitSHAResult):
		if sources.defaultSource.Git == nil {
			sources.defaultSource.Git = &build.GitSourceResult{}
		}

		// Source name is default as `spec.source` has no name field
		sources.defaultSource.Name = defaultSourceName
		sources.defaultSource.Git.CommitSha = result.Value
	case generateSourceResultName(defaultSourceName, commitAuthorResult):
		if sources.defaultSource.Git == nil {
			sources.defaultSource.Git = &build.GitSourceResult{}
		}

		// Source name is default as `spec.source` has no name field
		sources.defaultSource.Name = defaultSourceName
		sources.defaultSource.Git.CommitAuthor = result.Value
	case generateSourceResultName(defaultSourceName, bundleImageDigestResult):
		if sources.defaultSource.Bundle == nil {
			sources.bundleSource.Bundle = &build.BundleSourceResult{}
		}

		// Source name is default as `spec.source` has no name field
		sources.bundleSource.Name = defaultSourceName
		sources.bundleSource.Bundle.Digest = result.Value
	case generateOutputResultName(imageDigestResult):
		buildRun.Status.Output.Digest = result.Value
	case generateOutputResultName(imageSizeResult):
		buildRun.Status.Output.Size = result.Value
	}
}

func generateSourceResultName(source string, resultName string) string {
	return fmt.Sprintf("%s-source-%s-%s", prefixParamsResultsVolumes, defaultSourceName, resultName)
}

func generateOutputResultName(resultName string) string {
	return fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultName)
}
